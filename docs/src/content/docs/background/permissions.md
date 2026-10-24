---
title: Permissions
description: How permission bits are stored and computed.
---

Permissions are a 64-bit mask. On the wire Fluxer sends them as decimal strings, because JSON numbers can't hold all 64 bits. `fluxer.Permissions` takes care of that in both directions.

```go
p := fluxer.PermViewChannel | fluxer.PermSendMessages
p.Has(fluxer.PermSendMessages)                    // true
p.Missing(fluxer.PermSendMessages | fluxer.PermEmbedLinks) // PermEmbedLinks
fmt.Println(p)                                    // VIEW_CHANNEL|SEND_MESSAGES
```

## Computing a member's permissions

`fluxer.ChannelPermissions` follows the same order as Fluxer:

1. The guild owner has every permission.
2. Start with the permissions of the everyone role, whose ID equals the guild ID.
3. Add the permissions of every role the member has. With `ADMINISTRATOR`, the result is every permission and overwrites don't apply.
4. In a channel, apply the everyone overwrite, then all role overwrites together, then the member's own overwrite. An allow on one of the member's roles beats a deny on another.

```go
perms := fluxer.ChannelPermissions(guild, userID, member.Roles, channel)
```

Threads have no overwrites of their own. Use the parent channel; `cache.Permissions` does that for you.

## Role hierarchy

Being allowed to manage roles is not enough to manage every role. A member can only change roles below their own highest role, and only kick, ban or time out members whose highest role is lower. On equal positions the role with the smaller ID ranks higher. The owner is above everyone.

```go
if !cache.CanManageMember(guildID, cache.Self().ID, targetID) {
	return errors.New("my role is not high enough to do that")
}
```

## Things Fluxer adds

- `PermPinMessages` is separate from `PermManageMessages`.
- `PermBypassSlowmode` lets a member ignore slowmode.
- `PermViewChannelMembers` controls who sees the member list of a channel.
- In guilds that require two-factor authentication, moderation permissions only work for accounts with 2FA. A bot whose owner has no 2FA gets `TWO_FACTOR_REQUIRED`.
