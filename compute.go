package fluxer

import "slices"

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
