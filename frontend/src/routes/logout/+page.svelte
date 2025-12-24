<script lang="ts">
	import { enhance } from '$app/forms';

	let formElement: HTMLFormElement;

	$effect(() => {
		// Auto-submit the form when the page loads
		formElement?.requestSubmit();
	});
</script>

<form method="POST" bind:this={formElement} use:enhance>
	<p>Logging out...</p>
</form>
