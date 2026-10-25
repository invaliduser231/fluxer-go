package fluxer

// MemberListGroup is a group header in a member list: a hoisted role ID,
// online or offline, with the number of members in it.
type MemberListGroup struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// MemberListMember is a member in a member list together with its presence.
// The presence has status offline when the member is not visibly online.
type MemberListMember struct {
	Member
	Presence *Presence `json:"presence,omitempty"`
}

// MemberListItem is one position of a member list. Exactly one of Group and
// Member is set.
type MemberListItem struct {
	Group  *MemberListGroup  `json:"group,omitempty"`
	Member *MemberListMember `json:"member,omitempty"`
}

// MemberListOp replaces the inclusive index range Range of a member list with
// Items. Op is SYNC.
type MemberListOp struct {
	Op    string           `json:"op"`
	Range [2]int           `json:"range"`
	Items []MemberListItem `json:"items"`
}

// GuildMemberListUpdate is the GUILD_MEMBER_LIST_UPDATE dispatch, sent for the
// member list ranges subscribed with Session.Subscribe. ID is the channel ID of
// the list.
type GuildMemberListUpdate struct {
	GuildID     string            `json:"guild_id"`
	ID          string            `json:"id"`
	ChannelID   string            `json:"channel_id,omitempty"`
	MemberCount int               `json:"member_count"`
	OnlineCount int               `json:"online_count"`
	Groups      []MemberListGroup `json:"groups"`
	Ops         []MemberListOp    `json:"ops"`
}

// ThreadMemberListItem is a member of a thread member list with its presence.
type ThreadMemberListItem struct {
	ThreadMember
	Presence *Presence `json:"presence,omitempty"`
}

// ThreadMemberListUpdate is the THREAD_MEMBER_LIST_UPDATE dispatch with the full
// member list, at most 1,000 members, of a thread subscribed through
// GuildSubscription.ThreadMemberLists. Resuming does not replay it.
type ThreadMemberListUpdate struct {
	GuildID  string                 `json:"guild_id"`
	ThreadID string                 `json:"thread_id"`
	Members  []ThreadMemberListItem `json:"members"`
}

// EventName returns the gateway event name GUILD_MEMBER_LIST_UPDATE.
func (GuildMemberListUpdate) EventName() string { return "GUILD_MEMBER_LIST_UPDATE" }

// EventName returns the gateway event name THREAD_MEMBER_LIST_UPDATE.
func (ThreadMemberListUpdate) EventName() string { return "THREAD_MEMBER_LIST_UPDATE" }

func init() {
	registerEvents(func() Event { return &GuildMemberListUpdate{} }, func() Event { return &ThreadMemberListUpdate{} })
}
