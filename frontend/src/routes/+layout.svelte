<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import Header from '$lib/components/header/Header.svelte';
	import Footer from '$lib/components/footer/Footer.svelte';
	import type { LayoutData } from './$types';
	import type { Snippet } from 'svelte';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import { page } from '$app/state';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import AppSidebar from '$lib/components/app-sidebar.svelte';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';

	let { children, data }: { children: Snippet; data: LayoutData } = $props();

	// Check if current route is an admin route
	let isAdminRoute = $derived(page.route.id?.startsWith('/(admin)'));
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

<Toaster position="top-center" richColors />

{#if isAdminRoute}
	<!-- Admin routes: no header/footer, full layout control for sidebar -->
	<Sidebar.Provider class="bg-white">
		<AppSidebar />
		<Sidebar.Inset>
			<header class="flex h-16 shrink-0 items-center gap-2 px-4">
				<Sidebar.Trigger class="-ms-1" />
				<Separator orientation="vertical" class="me-2 data-[orientation=vertical]:h-4" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						<Breadcrumb.Item class="hidden md:block">
							<Breadcrumb.Link class="capitalize" href="##"
								>{page.route?.id?.split('/').at(-1)}</Breadcrumb.Link
							>
						</Breadcrumb.Item>
					</Breadcrumb.List>
				</Breadcrumb.Root>
			</header>
			{@render children()}
		</Sidebar.Inset>
	</Sidebar.Provider>
{:else}
	<!-- Regular routes: with header and footer -->
	<div class="flex h-full min-h-dvh flex-col">
		<Header isLoggedIn={data.isLoggedIn} />
		{@render children()}
		<div class="flex grow flex-col justify-end">
			<Footer />
		</div>
	</div>
{/if}
