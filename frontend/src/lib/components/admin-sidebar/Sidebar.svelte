<script lang="ts">
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import { useSidebar } from '$lib/components/ui/sidebar/index.js';
	import type { ComponentProps } from 'svelte';
	import { Settings, Users, ShieldUser } from '@lucide/svelte';
	import { resolve } from '$app/paths';

	let { ref = $bindable(null), ...restProps }: ComponentProps<typeof Sidebar.Root> = $props();
	const sidebar = useSidebar();

	const closeMobileSidebar = () => {
		if (sidebar.isMobile) {
			sidebar.setOpenMobile(false);
		}
	};
</script>

<Sidebar.Root bind:ref variant="floating" {...restProps}>
	<Sidebar.Header>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton size="lg">
					{#snippet child({ props })}
						<a href="##" {...props}>
							<div
								class="flex aspect-square size-8 items-center justify-center rounded-lg bg-blue-500 text-sidebar-primary-foreground"
							>
								<ShieldUser strokeWidth={1.5} />
							</div>
							<div class="flex flex-col gap-0.5 leading-none">
								<span class="font-inter font-medium">Admin Dashboard</span>
							</div>
						</a>
					{/snippet}
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					<Sidebar.MenuButton onclick={closeMobileSidebar}>
						<Settings class="inline-block " size={18} />
						<a href={resolve('/admin/settings')} class="w-full font-inter">Settings</a>
					</Sidebar.MenuButton>
					<Sidebar.MenuButton onclick={closeMobileSidebar}>
						<Users class="inline-block" size={18} />
						<a href={resolve('/admin/users')} class="w-full font-inter"> Users </a>
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		</Sidebar.Group>
	</Sidebar.Content>
</Sidebar.Root>
