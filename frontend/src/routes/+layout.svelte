<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import Header from '$lib/components/header/Header.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import Footer from '$lib/components/footer/Footer.svelte';
	import type { LayoutData } from './$types';
	import type { Snippet } from 'svelte';

	let { children, data }: { children: Snippet; data: LayoutData } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />

	<!-- Google Fonts -->
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<Dialog.Root>
	<div class="flex h-full min-h-dvh flex-col">
		<Header isLoggedIn={data.isLoggedIn} />
		{@render children()}
		<div class="flex grow flex-col justify-end">
			<Footer />
		</div>
	</div>
</Dialog.Root>
