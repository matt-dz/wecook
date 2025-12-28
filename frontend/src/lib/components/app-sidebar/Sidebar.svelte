<script lang="ts">
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import type { ComponentProps } from 'svelte';
	import { resolve } from '$app/paths';
	import { useSidebar } from '$lib/components/ui/sidebar/index.js';

	type Props = ComponentProps<typeof Sidebar.Root> & {
		loggedIn: boolean;
	};

	let { ref = $bindable(null), loggedIn = $bindable(), ...restProps }: Props = $props();
	const sidebar = useSidebar();

	const closeSidebar = () => {
		if (sidebar.isMobile) {
			sidebar.setOpenMobile(false);
		} else {
			sidebar.setOpen(false);
		}
	};
</script>

<Sidebar.Root bind:ref {...restProps}>
	<Sidebar.Header>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<button onclick={closeSidebar}>
					<a href={resolve('/')}>WeCook</a>
				</button>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					{#if loggedIn}
						<Sidebar.MenuButton onclick={closeSidebar}>
							<a href={resolve('/(user)/home')} class="w-full font-inter">My Recipes</a>
						</Sidebar.MenuButton>
						<Sidebar.MenuButton onclick={closeSidebar}>
							<a href={resolve('/(user)/profile')} class="w-full font-inter">Profile</a>
						</Sidebar.MenuButton>
						<Sidebar.MenuButton onclick={closeSidebar}>
							<a href={resolve('/logout')} class="w-full font-inter">Logout</a>
						</Sidebar.MenuButton>
					{:else}
						<Sidebar.MenuButton onclick={closeSidebar}>
							<a href={resolve('/login')} class="w-full font-inter">Login</a>
						</Sidebar.MenuButton>
					{/if}
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		</Sidebar.Group>
	</Sidebar.Content>
</Sidebar.Root>
