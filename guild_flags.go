package fluxer

// Verification levels for Guild.VerificationLevel.
const (
	VerificationNone   = 0
	VerificationLow    = 1
	VerificationMedium = 2
	VerificationHigh   = 3
)

// MFA levels for Guild.MFALevel. MFAElevated requires moderators to have two-factor
// authentication enabled.
const (
	MFANone     = 0
	MFAElevated = 1
)

// Explicit content filter settings for Guild.ExplicitContentFilter.
const (
	ContentFilterDisabled            = 0
	ContentFilterMembersWithoutRoles = 1
	ContentFilterAllMembers          = 2
)

// Default notification settings for Guild.DefaultMessageNotifications.
const (
	NotifyAllMessages  = 0
	NotifyOnlyMentions = 1
)

// NSFW levels for the legacy Guild.NSFWLevel. Guild.NSFW is the current setting.
const (
	NSFWLevelSafe          = 0
	NSFWLevelAgeRestricted = 3
)

// Splash card alignments for Guild.SplashCardAlignment.
const (
	SplashCenter = 0
	SplashLeft   = 1
	SplashRight  = 2
)

// SystemSuppressJoinNotifications is the bit in Guild.SystemChannelFlags that stops join
// messages in the system channel.
const SystemSuppressJoinNotifications = 1 << 0

// Operation bits for Guild.DisabledOperations. A set bit turns that operation off for the
// whole guild.
const (
	OperationPushNotifications = 1 << 0
	OperationEveryoneMentions  = 1 << 1
	OperationTypingEvents      = 1 << 2
	OperationInstantInvites    = 1 << 3
	OperationSendMessage       = 1 << 4
	OperationReactions         = 1 << 5
	OperationMemberListUpdates = 1 << 6
)

// Guild feature flags as found in Guild.Features.
const (
	FeatureAnimatedIcon                   = "ANIMATED_ICON"
	FeatureAnimatedBanner                 = "ANIMATED_BANNER"
	FeatureAudioBitrate128                = "AUDIO_BITRATE_128_KBPS"
	FeatureAudioBitrate256                = "AUDIO_BITRATE_256_KBPS"
	FeatureAudioBitrate384                = "AUDIO_BITRATE_384_KBPS"
	FeatureBanner                         = "BANNER"
	FeatureCloneEmojiDisabled             = "CLONE_EMOJI_DISABLED"
	FeatureCloneEmojiEnabled              = "CLONE_EMOJI_ENABLED"
	FeatureCloneStickerDisabled           = "CLONE_STICKER_DISABLED"
	FeatureCloneStickerEnabled            = "CLONE_STICKER_ENABLED"
	FeatureDetachedBanner                 = "DETACHED_BANNER"
	FeatureInviteSplash                   = "INVITE_SPLASH"
	FeatureInvitesDisabled                = "INVITES_DISABLED"
	FeatureRaidDetected                   = "RAID_DETECTED"
	FeatureTextChannelFlexibleNames       = "TEXT_CHANNEL_FLEXIBLE_NAMES"
	FeatureHideOwnerCrown                 = "HIDE_OWNER_CROWN"
	FeatureMoreEmoji                      = "MORE_EMOJI"
	FeatureMoreStickers                   = "MORE_STICKERS"
	FeatureUnlimitedEmoji                 = "UNLIMITED_EMOJI"
	FeatureUnlimitedStickers              = "UNLIMITED_STICKERS"
	FeatureExpressionPurgeAllowed         = "EXPRESSION_PURGE_ALLOWED"
	FeatureVanityURL                      = "VANITY_URL"
	FeatureDiscoverable                   = "DISCOVERABLE"
	FeaturePartnered                      = "PARTNERED"
	FeatureVerified                       = "VERIFIED"
	FeatureVIPVoice                       = "VIP_VOICE"
	FeatureVoiceE2EE                      = "VOICE_E2EE"
	FeatureUnavailableForEveryone         = "UNAVAILABLE_FOR_EVERYONE"
	FeatureUnavailableForEveryoneButStaff = "UNAVAILABLE_FOR_EVERYONE_BUT_STAFF"
	FeatureUnavailableHidden              = "UNAVAILABLE_HIDDEN"
	FeatureVisionary                      = "VISIONARY"
	FeatureLargeGuildOverride             = "LARGE_GUILD_OVERRIDE"
	FeatureVeryLargeGuild                 = "VERY_LARGE_GUILD"
	FeatureAnnouncementChannelsDisabled   = "ANNOUNCEMENT_CHANNELS_DISABLED"
	FeatureVisitsEnabled                  = "VISITS_ENABLED"
)
