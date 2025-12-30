<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index.js';

	let { email, isCurrentUser }: { email: string; isCurrentUser: boolean } = $props();
</script>

<div class="flex items-center gap-2">
	<span class="font-inter">{email}</span>
	{#if isCurrentUser}
		<Badge variant="secondary" class="bg-blue-50 text-blue-700 hover:bg-blue-100">You</Badge>
	{/if}
</div>
