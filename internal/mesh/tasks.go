package mesh

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/dukedelaet/crush-bot/internal/config"
	"github.com/dukedelaet/crush-bot/internal/crush"
	"github.com/dukedelaet/crush-bot/internal/envelope"
	"github.com/dukedelaet/crush-bot/internal/roster"
	"github.com/dukedelaet/crush-bot/internal/task"
)

func tasksEnabled(id Identity) bool {
	if id.Root == "" {
		return false
	}
	cfg, err := config.Load(config.Paths{ConfigFile: id.Root + "/config.yaml"})
	if err != nil {
		return false
	}
	return cfg.Experimental.Tasks
}

func AssignTask(id Identity, target, title, body, priority, key string) CallResult {
	if err := id.Validate(); err != nil {
		return failReason(err)
	}
	if !tasksEnabled(id) {
		return CallResult{Reason: "missing_config", Error: "experimental.tasks is false"}
	}
	target = strings.TrimPrefix(target, "@")
	if utf8.RuneCountInString(body) > MessageMaxChars {
		return CallResult{Reason: "message_too_long", Error: "body too long"}
	}
	if !roster.Exists(id.Root, target) {
		return CallResult{Reason: "unknown_bot", Error: "unknown bot " + target}
	}
	if target == id.Bot {
		return CallResult{Reason: "self_message", Error: "cannot assign to self"}
	}
	turn, err := crush.ReadTurn(id.BotHome())
	if err != nil {
		return CallResult{Reason: "missing_config", Error: "no turn context"}
	}
	hop := turn.InboundHop + 1
	if hop > turn.MaxHops && turn.MaxHops > 0 {
		return CallResult{Reason: "hop_limit", Error: "hop exceeds max"}
	}
	if err := bumpSends(id.BotHome(), turn.MaxSends); err != nil {
		return failReason(err)
	}
	t, dup, err := task.Assign(id.Root, task.AssignOpts{
		From: id.Bot, To: target, Title: title, Body: body,
		Priority: priority, Key: key, Hop: hop,
	})
	if err != nil {
		return CallResult{Reason: "runtime_offline", Error: err.Error()}
	}
	st := "queued"
	if DaemonLive(id.Root) {
		st = "sent"
	}
	if dup {
		st = "queued"
	}
	return CallResult{Status: st, ID: t.ID, To: target}
}

func TaskList(id Identity) (any, CallResult) {
	if err := id.Validate(); err != nil {
		return nil, failReason(err)
	}
	if !tasksEnabled(id) {
		return nil, CallResult{Reason: "missing_config", Error: "experimental.tasks is false"}
	}
	list, err := task.ListFor(id.Root, id.Bot)
	if err != nil {
		return nil, CallResult{Reason: "unknown", Error: err.Error()}
	}
	return list, CallResult{Status: "ok"}
}

func TaskComplete(id Identity, taskID, result string) CallResult {
	return finishTask(id, taskID, "done", result, "")
}

func TaskFail(id Identity, taskID, reason, errMsg string) CallResult {
	status := "failed"
	if reason == "need_human" {
		status = "blocked"
	}
	return finishTask(id, taskID, status, errMsg, reason)
}

func finishTask(id Identity, taskID, status, result, reason string) CallResult {
	if err := id.Validate(); err != nil {
		return failReason(err)
	}
	if !tasksEnabled(id) {
		return CallResult{Reason: "missing_config", Error: "experimental.tasks is false"}
	}
	t, owner, err := task.Find(id.Root, taskID)
	if err != nil {
		return CallResult{Reason: "unknown", Error: err.Error()}
	}
	t.Status = status
	if result != "" {
		t.Result = &result
	}
	if reason != "" {
		t.Reason = &reason
	}
	if err := task.Save(id.Root, owner, t); err != nil {
		return CallResult{Reason: "runtime_offline", Error: err.Error()}
	}
	if t.From != "" && t.From != "user" && roster.Exists(id.Root, t.From) {
		hop := t.Hop + 1
		body := fmt.Sprintf("task %s is %s", t.ID, status)
		if result != "" {
			body += "\n" + result
		}
		_, _ = envelope.WritePending(roster.Home(id.Root, t.From), envelope.Envelope{
			Kind: "receipt", From: id.Bot, To: t.From, Hop: hop, TaskID: &t.ID,
			Body: body, Attribution: "Task receipt from @" + id.Bot + ":",
			Trace: []string{t.From, id.Bot},
		})
	}
	if t.ParentID != nil {
		wakeParent(id.Root, t)
	}
	st := "queued"
	if DaemonLive(id.Root) {
		st = "sent"
	}
	return CallResult{Status: st, ID: t.ID}
}

func TaskDelegate(id Identity, taskID, target, title, body string) CallResult {
	if err := id.Validate(); err != nil {
		return failReason(err)
	}
	if !tasksEnabled(id) {
		return CallResult{Reason: "missing_config", Error: "experimental.tasks is false"}
	}
	parent, owner, err := task.Find(id.Root, taskID)
	if err != nil {
		return CallResult{Reason: "unknown", Error: err.Error()}
	}
	target = strings.TrimPrefix(target, "@")
	if !roster.Exists(id.Root, target) {
		return CallResult{Reason: "unknown_bot", Error: "unknown bot " + target}
	}
	hop := parent.Hop + 1
	child, _, err := task.Assign(id.Root, task.AssignOpts{
		From: id.Bot, To: target, Title: title, Body: body,
		Hop: hop, ParentID: &parent.ID, Key: parent.ID + ":" + target + ":" + title,
	})
	if err != nil {
		return CallResult{Reason: "runtime_offline", Error: err.Error()}
	}
	parent.Status = "waiting_child"
	_ = task.Save(id.Root, owner, parent)
	st := "queued"
	if DaemonLive(id.Root) {
		st = "sent"
	}
	return CallResult{Status: st, ID: child.ID, To: target}
}

func wakeParent(root string, child task.Task) {
	if child.ParentID == nil {
		return
	}
	p, owner, err := task.Find(root, *child.ParentID)
	if err != nil {
		return
	}
	if p.Status != "waiting_child" {
		return
	}
	p.Status = "queued"
	_ = task.Save(root, owner, p)
	pid := p.ID
	_, _ = envelope.WritePending(roster.Home(root, p.To), envelope.Envelope{
		Kind: "task", From: child.To, To: p.To, Hop: p.Hop, TaskID: &pid,
		Body:        fmt.Sprintf("child %s is %s", child.ID, child.Status),
		Attribution: "Child task finished",
	})
}
