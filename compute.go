package fluxer

import "slices"

// BasePermissions computes the guild-wide permissions of a user with the given
// roles by combining the @everyone role, whose ID is the guild ID, with every
// listed role. The guild owner and anyone with PermAdministrator get PermAll.
// Role IDs missing from g.Roles are ignored.
func BasePermissions(g Guild, userID string, roleIDs []string) Permissions {
	if userID != "" && userID == g.OwnerID {
		return PermAll
	}
	var perms Permissions
	for _, r := range g.Roles {
		if r.ID == g.ID {
			perms |= r.Permissions
			break
		}
	}
	for _, id := range roleIDs {
		for _, r := range g.Roles {
			if r.ID == id {
				perms |= r.Permissions
				break
			}
		}
	}
	if perms.Has(PermAdministrator) {
		return PermAll
	}
	return perms
}

// ChannelPermissions computes a user's permissions in ch by applying its
// overwrites to BasePermissions: first the @everyone overwrite, then the
// combined overwrites of the user's roles, then the overwrite for the user.
// Overwrites do not affect PermAll.
func ChannelPermissions(g Guild, userID string, roleIDs []string, ch Channel) Permissions {
	perms := BasePermissions(g, userID, roleIDs)
	if perms == PermAll {
		return perms
	}
	var allow, deny Permissions
	var member *Overwrite
	for i, o := range ch.PermissionOverwrites {
		switch {
		case o.Type == OverwriteRole && o.ID == g.ID:
			perms = perms&^o.Deny | o.Allow
		case o.Type == OverwriteRole && slices.Contains(roleIDs, o.ID):
			allow |= o.Allow
			deny |= o.Deny
		case o.Type == OverwriteMember && o.ID == userID:
			member = &ch.PermissionOverwrites[i]
		}
	}
	perms = perms&^deny | allow
	if member != nil {
		perms = perms&^member.Deny | member.Allow
	}
	return perms
}

// MemberPermissions returns the permissions of m in g, or in ch when ch is
// non-nil. A member without User is never treated as the guild owner.
func MemberPermissions(g Guild, m Member, ch *Channel) Permissions {
	var userID string
	if m.User != nil {
		userID = m.User.ID
	}
	if ch == nil {
		return BasePermissions(g, userID, m.Roles)
	}
	return ChannelPermissions(g, userID, m.Roles, *ch)
}
