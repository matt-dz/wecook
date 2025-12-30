<script lang="ts">
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import type { PageProps } from './$types';
	import { updatePreferences } from '$lib/admin';
	import Button from '$lib/components/button/Button.svelte';
	import fetch from '$lib/http';
	import { parseError } from '$lib/errors/api';
	import { HTTPError } from 'ky';
	import { toast } from 'svelte-sonner';

	let { data }: PageProps = $props();

	let allowPublicSignup = $state(data.preferences.allow_public_signup);
	let saving = $state(false);

	const handleSave = async () => {
		try {
			saving = true;
			await updatePreferences(fetch, {
				allow_public_signup: allowPublicSignup
			});
			toast.success('Saved preferences successfully.');
		} catch (e) {
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				if (err.success) {
					console.error('failed to save preferences', err.data);
				}
			}
			console.error('failed to save preferences', e);
			toast.error('Failed to save preferences');
		} finally {
			saving = false;
		}
	};
</script>

<div class="mt-12 flex justify-center px-6">
	<div class="mt-8 flex w-full max-w-3xl flex-col items-center gap-4">
		<div class="flex items-start gap-2">
			<Switch id="public-signup" bind:checked={allowPublicSignup} disabled={saving} />
			<div class="space-y-1">
				<Label for="public-signup" class="font-inter">Public Signup</Label>
				<p class="font-inter text-sm font-light text-gray-400">
					If disabled (default), users must receive an invitation to signup.
				</p>
			</div>
		</div>

		<Button disabled={saving} className="rounded-md text-sm mt-12" onclick={handleSave}>Save</Button
		>
	</div>
</div>
