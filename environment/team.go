package environment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/antgroup/aievo/schema"
)

type SubscribeMode int

var (
	ErrMissingLeader = errors.New("leader agent is not set")
)

const (
	// DefaultSubMode 存在leader agent的时候，则使用leader sub,否则ALL sub
	DefaultSubMode SubscribeMode = iota
	LeaderSubMode
	ALLSubMode
	CustomSubMode
)

type Team struct {
	Members    []schema.Agent
	Leader     schema.Agent
	Subscribes []schema.Subscribe
	SubMode    SubscribeMode
}

func NewTeam() *Team {
	return &Team{
		Members:    []schema.Agent{},
		Subscribes: []schema.Subscribe{},
		SubMode:    DefaultSubMode,
	}
}

func (t *Team) InitSubRelation() error {
	// 根据订阅模式填充订阅关系
	switch t.SubMode {
	case DefaultSubMode:
		if t.Leader != nil {
			t.buildLeaderSubRelation()
		} else {
			t.buildAllSubRelation()
		}
	case LeaderSubMode:
		if t.Leader != nil {
			return ErrMissingLeader
		}
		t.buildLeaderSubRelation()
	case ALLSubMode:
		t.buildAllSubRelation()
	case CustomSubMode:
	}
	t.deduplicateSub()
	return nil
}

func (t *Team) Member(name string) schema.Agent {
	for _, a := range t.Members {
		if strings.EqualFold(a.Name(), name) {
			return a
		}
	}
	return nil
}

func (t *Team) AddMembers(members ...schema.Agent) {
	t.Members = append(t.Members, members...)
}

func (t *Team) GetSubMembers(_ context.Context,
	subscribed schema.Agent) []schema.Agent {
	members := make([]schema.Agent, 0)
	for _, subscribe := range t.Subscribes {
		if subscribe.Subscribed.Name() == subscribed.Name() &&
			subscribe.Subscriber.Name() != subscribed.Name() {
			// 仅考虑组内成员
			for _, member := range t.Members {
				if subscribe.Subscriber.Name() == member.Name() {
					members = append(members, subscribe.Subscriber)
					break
				}
			}
		}
	}
	return members
}

func (t *Team) GetMsgSubMembers(msg *schema.Message) (subscribers []string) {
	for _, subscribe := range t.Subscribes {
		if strings.EqualFold(subscribe.Subscribed.Name(), msg.Sender) {
			if subscribe.Condition != "" && subscribe.Condition != msg.Condition {
				continue
			}
			for _, member := range t.Members {
				if subscribe.Subscriber.Name() == member.Name() {
					subscribers = append(subscribers, subscribe.Subscriber.Name())
					break
				}
			}
		}
	}
	return subscribers
}

func (t *Team) RemoveMembers(names []string) {
	for _, name := range names {
		for i, member := range t.Members {
			if strings.EqualFold(member.Name(),
				strings.TrimSpace(name)) {
				t.Members = append(t.Members[:i], t.Members[i+1:]...)
				break
			}
		}
	}
}

func (t *Team) buildLeaderSubRelation() {
	for _, a := range t.Members {
		if a.Name() == t.Leader.Name() {
			continue
		}
		t.Subscribes = append(t.Subscribes,
			schema.Subscribe{
				Subscribed: a,
				Subscriber: t.Leader,
			}, schema.Subscribe{
				Subscribed: t.Leader,
				Subscriber: a,
			})
	}
}

func (t *Team) buildAllSubRelation() {
	for _, a := range t.Members {
		for _, tmp := range t.Members {
			if a.Name() == tmp.Name() {
				continue
			}
			t.Subscribes = append(t.Subscribes,
				schema.Subscribe{
					Subscribed: a,
					Subscriber: tmp,
				})
		}
	}
}

func (t *Team) deduplicateSub() {
	m := make(map[string]struct{})
	subs := make([]schema.Subscribe, 0, len(t.Subscribes))
	for _, sub := range t.Subscribes {
		if sub.Subscribed == sub.Subscriber {
			continue
		}
		k := fmt.Sprintf("%s-%s-%s",
			sub.Subscriber, sub.Subscribed, sub.Condition)
		if _, ok := m[k]; !ok {
			subs = append(subs, sub)
			m[k] = struct{}{}
		}
	}
	t.Subscribes = subs
}
