package messaging

import "testing"

func TestSubjectBuilder(t *testing.T) {
    sb := NewSubjectBuilder()

    in := sb.WorkflowIn("wf-1")
    out := sb.WorkflowOut("wf-1")
    if in != "workflows.wf-1.in" {
        t.Fatalf("unexpected workflow in: %s", in)
    }
    if out != "workflows.wf-1.out" {
        t.Fatalf("unexpected workflow out: %s", out)
    }

    ai := sb.AgentIn("agent-1")
    ao := sb.AgentOut("agent-1")
    if ai != "agents.agent-1.in" {
        t.Fatalf("unexpected agent in: %s", ai)
    }
    if ao != "agents.agent-1.out" {
        t.Fatalf("unexpected agent out: %s", ao)
    }
}
