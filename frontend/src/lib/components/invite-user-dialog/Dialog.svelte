<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import fetch from '$lib/http';
	import { inviteUser } from '$lib/users';
	import { toast } from 'svelte-sonner';
	import Button from '../button/Button.svelte';
	import Input from '../input/Input.svelte';
	import { HTTPError } from 'ky';
	import { parseError } from '$lib/errors/api';
	import { Spinner } from '$lib/components/ui/spinner/index.js';

	let email = $state('');
	let inviting = $state(false);

	const handleInvite = async () => {
		try {
			inviting = true;
			await inviteUser(fetch, { email });
			toast.success('Invite sent successfully.');
		} catch (e) {
			toast.error('Failed to invite user.');
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				if (err.success) {
					console.error(err.data);
				}
			}
			console.error(e);
		} finally {
			inviting = false;
		}
	};
</script>

<Dialog.Content>
	<form>
		<Dialog.Title class="mb-2 font-inter">Invite User</Dialog.Title>
		<Dialog.Description class="font-inter"
			>The user will receive an invitation email to signup for the WeCook platform.</Dialog.Description
		>

		<div class="mt-4 flex flex-col gap-1">
			<label for="email" class="font-inter">Email</label>
			<Input
				bind:value={email}
				name="email"
				type="email"
				placeholder="email"
				autocapitalize="none"
				autocorrect="off"
				spellcheck="false"
				required
				className="w-full font-inter py-1.5 text-base"
			/>
		</div>

		<Dialog.Footer class="mt-4">
			<Dialog.Close>
				<Button>Cancel</Button>
			</Dialog.Close>
			<Button
				type="submit"
				onclick={handleInvite}
				disabled={inviting}
				className="from-blue-300 to-blue-200 hover:from-blue-200 hover:to-blue-100"
			>
				{#if inviting}
					<Spinner />
				{:else}
					Invite
				{/if}
			</Button>
		</Dialog.Footer>
	</form>
</Dialog.Content>
