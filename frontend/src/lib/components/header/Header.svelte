<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import clsx from 'clsx';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';

	interface Props {
		isLoggedIn: boolean;
	}

	let { isLoggedIn }: Props = $props();
</script>

<header class="relative z-10 flex justify-center px-6 pt-8">
	<div class="flex w-full max-w-5xl items-center justify-between">
		<a href={resolve('/')} class="text-3xl">WeCook</a>

		<nav class="hidden gap-4 sm:flex">
			{#if isLoggedIn}
				<a
					href={resolve('/home')}
					class={clsx(
						'underline-offset-2 hover:underline',
						page.route.id === '/(user)/home' && 'underline'
					)}>My Recipes</a
				>
				<a
					href={resolve('/profile')}
					class={clsx(
						'underline-offset-2 hover:underline',
						page.route.id === '/(user)/profile' && 'underline'
					)}>Profile</a
				>
				<a href={resolve('/logout')} class="underline-offset-2 hover:underline">Logout</a>
			{:else}
				<a href={resolve('/login')} class="underline-offset-2 hover:underline">Login</a>
			{/if}
		</nav>
		<Sidebar.Trigger class="sm:hidden" />
	</div>
</header>
