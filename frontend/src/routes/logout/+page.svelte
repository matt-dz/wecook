<script lang="ts">
	import { logout } from '$lib/auth';
	import { parseError } from '$lib/errors/api';
	import fetch from '$lib/http';
	import { goto, invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { HTTPError } from 'ky';
	import { toast } from 'svelte-sonner';
	let formElement: HTMLFormElement;

	const handleLogout = async () => {
		try {
			await logout(fetch);
			await invalidateAll();
			goto(resolve('/login'));
		} catch (e) {
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				if (err.success) {
					console.error('failed to logout', e);
				}
			}
			console.error('failed to logout', e);
			toast.error('Failed to logout.');
		}
	};

	$effect(() => {
		// Auto-submit the form when the page loads
		formElement?.requestSubmit();
	});
</script>

<form bind:this={formElement} onsubmit={handleLogout}>
	<p>Logging out...</p>
</form>
