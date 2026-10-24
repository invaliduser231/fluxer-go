import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';

export default defineConfig({
	site: 'https://invaliduser231.github.io',
	base: '/fluxer-go',
	integrations: [
		starlight({
			title: 'fluxer-go',
			description: 'Go library for the Fluxer API.',
			logo: { src: './src/assets/logo.svg' },
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/invaliduser231/fluxer-go' },
				{ icon: 'comment', label: 'Community on Fluxer', href: 'https://fluxer.gg/a3ULk923' },
			],
			editLink: { baseUrl: 'https://github.com/invaliduser231/fluxer-go/edit/main/docs/' },
			lastUpdated: true,
			customCss: ['./src/styles/custom.css'],
			sidebar: [
				{
					label: 'Start',
					items: [
						{ label: 'Overview', link: '/' },
						{ slug: 'start/installation' },
						{ slug: 'start/first-bot' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ slug: 'guides/messages' },
						{ slug: 'guides/events' },
						{ slug: 'guides/gateway' },
						{ slug: 'guides/sharding' },
						{ slug: 'guides/cache' },
						{ slug: 'guides/redis' },
						{ slug: 'guides/voice' },
						{ slug: 'guides/webhooks' },
						{ slug: 'guides/oauth2' },
						{ slug: 'guides/instances' },
						{ slug: 'guides/testing' },
					],
				},
				{
					label: 'Background',
					items: [
						{ slug: 'background/errors-and-rate-limits' },
						{ slug: 'background/resuming' },
						{ slug: 'background/permissions' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'fluxer', link: 'https://pkg.go.dev/github.com/invaliduser231/fluxer-go', attrs: { target: '_blank' } },
						{ label: 'state', link: 'https://pkg.go.dev/github.com/invaliduser231/fluxer-go/state', attrs: { target: '_blank' } },
						{ label: 'state/redis', link: 'https://pkg.go.dev/github.com/invaliduser231/fluxer-go/state/redis', attrs: { target: '_blank' } },
						{ label: 'voice', link: 'https://pkg.go.dev/github.com/invaliduser231/fluxer-go/voice', attrs: { target: '_blank' } },
						{ label: 'fluxertest', link: 'https://pkg.go.dev/github.com/invaliduser231/fluxer-go/fluxertest', attrs: { target: '_blank' } },
						{ label: 'Changelog', link: 'https://github.com/invaliduser231/fluxer-go/blob/main/CHANGELOG.md', attrs: { target: '_blank' } },
					],
				},
			],
		}),
	],
});
