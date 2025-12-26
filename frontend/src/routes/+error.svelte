<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { MoveLeft } from '@lucide/svelte';
</script>

<div class="mt-16 flex items-center justify-center">
	<div class="flex flex-col items-center">
		<img src="/error.png" alt="error" width={128} height={128} />
		<div class="mt-4 w-full max-w-3xl px-4">
			<h1 class="text-center text-xl">Uh-Oh! There was an error.</h1>
			<div class="mt-4 flex flex-col gap-2">
				{#if page.error?.code}
					<div>
						<h2 class="font-semibold">Code</h2>
						<p class="text-sm text-red-600">{page.error?.code}</p>
					</div>
				{/if}
				{#if page.error?.status}
					<div>
						<h2 class="font-semibold">Status</h2>
						<p class="text-sm text-red-600">{page.error?.status}</p>
					</div>
				{/if}
				{#if page.error?.errorId}
					<div>
						<h2 class="font-semibold">Error ID</h2>
						<p class="text-sm text-red-600">{page.error?.errorId}</p>
					</div>
				{/if}
				{#if page.error?.message}
					<div>
						<h2 class="font-semibold">Message</h2>
						<p class="text-sm text-red-600">{page.error?.message}</p>
					</div>
				{/if}
			</div>
		</div>
		<a class="mt-8 flex items-center gap-1 rounded-lg underline" href={resolve('/')}
			><MoveLeft strokeWidth={1.25} size={24} class="inline-block" />Home</a
		>
	</div>
</div>
