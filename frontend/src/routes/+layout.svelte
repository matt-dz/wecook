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
	import AdminSidebar from '$lib/components/admin-sidebar/Sidebar.svelte';
	import AppSidebar from '$lib/components/app-sidebar/Sidebar.svelte';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';

	let { children, data }: { children: Snippet; data: LayoutData } = $props();

	// Check if current route is an admin route
	let isAdminRoute = $derived(page.route.id?.startsWith('/(admin)'));
	let sidebarOpen = $state(false);
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

	<!-- Primary Meta Tags -->
	<title>WeCook</title>
	<meta name="title" content="WeCook - Self-Hosted Recipe Manager" />
	<meta
		name="description"
		content="A self-hosted recipe manager for organizing and sharing your favorite recipes. Create, edit, and publish recipes with ingredients, steps, and images."
	/>
	<meta
		name="keywords"
		content="recipe manager, self-hosted, recipe organizer, cooking, recipes, open source"
	/>
	<meta name="author" content="WeCook" />

	<!-- Open Graph / Facebook -->
	<meta property="og:type" content="website" />
	<meta property="og:url" content={page.url.href} />
	<meta property="og:title" content="WeCook - Self-Hosted Recipe Manager" />
	<meta
		property="og:description"
		content="A self-hosted recipe manager for organizing and sharing your favorite recipes."
	/>
	<meta property="og:site_name" content="WeCook" />

	<!-- Twitter -->
	<meta property="twitter:card" content="summary_large_image" />
	<meta property="twitter:url" content={page.url.href} />
	<meta property="twitter:title" content="WeCook - Self-Hosted Recipe Manager" />
	<meta
		property="twitter:description"
		content="A self-hosted recipe manager for organizing and sharing your favorite recipes."
	/>

	<!-- Canonical URL -->
	<link rel="canonical" href={page.url.href} />
</svelte:head>

<Toaster position="top-center" richColors />

{#if isAdminRoute}
	<!-- Admin routes: no header/footer, full layout control for sidebar -->
	<Sidebar.Provider class="bg-white">
		<AdminSidebar />
		<Sidebar.Inset>
			<header class="flex h-16 shrink-0 items-center gap-2 px-4">
				<Sidebar.Trigger class="-ms-1" />
				<Separator orientation="vertical" class="me-2 data-[orientation=vertical]:h-4" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						<Breadcrumb.Item>
							<Breadcrumb.Link class="font-inter capitalize" href="##"
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
	<Sidebar.Provider class="white" bind:open={sidebarOpen}>
		<AppSidebar
			loggedIn={data.isLoggedIn}
			side="right"
			variant="floating"
			collapsible="offcanvas"
			class="sm:invisible"
		/>
		<Sidebar.Inset>
			<main class="flex min-h-screen flex-col">
				<Header isLoggedIn={data.isLoggedIn} />
				<div class="grow">
					{@render children()}
				</div>
				<Footer />
			</main>
		</Sidebar.Inset>
	</Sidebar.Provider>
{/if}
