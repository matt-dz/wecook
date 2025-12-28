<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import Button from '../button/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { changePassword } from '$lib/users';
	import fetch from '$lib/http';
	import { HTTPError } from 'ky';
	import { parseError } from '$lib/errors/api';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import Error from '$lib/components/error/Error.svelte';
	import { toast } from 'svelte-sonner';

	const handleChangePassword = async () => {
		try {
			loading = true;
			await changePassword(fetch, { new_password: newPassword, current_password: currentPassword });
			open = false;
			toast.success('Changed password successfully');
		} catch (e) {
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				if (err.success) {
					console.error(err.data);
					error = err.data.message;
					return;
				}
			}
			console.error(e);
			toast.error('Failed to change password.');
		} finally {
			loading = false;
		}
	};

	interface Props {
		open?: boolean;
	}

	let { open = $bindable(false) }: Props = $props();

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmation = $state('');
	let loading = $state(false);
	let passwordsMatch = $derived.by(() => newPassword === confirmation);
	$effect(() => {
		if (newPassword !== confirmation && (newPassword.length > 0 || confirmation.length > 0)) {
			error = 'Passwords do not match.';
		} else {
			error = '';
		}
	});
	$effect(() => {
		if (!open) {
			currentPassword = '';
			newPassword = '';
			confirmation = '';
			loading = false;
		}
	});
	let error = $state('');
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Content>
			<Dialog.Header>
				<Dialog.Title class="font-inter">Change Password</Dialog.Title>
				<Dialog.Description class="font-inter">
					Enter your current password and your new password. You will not be able to login with your
					current password once it is changed.
				</Dialog.Description>
			</Dialog.Header>
			<div class="space-y-2">
				<Label class="font-inter" for="cpassword">Current Password</Label>
				<Input type="password" required bind:value={currentPassword} disabled={loading} />
			</div>
			<div class="space-y-2">
				<Label class="font-inter" for="cpassword">New Password</Label>
				<Input type="password" required bind:value={newPassword} disabled={loading} />
			</div>
			<div class="space-y-2">
				<Label class="font-inter" for="cpassword">Confirm New Password</Label>
				<Input type="password" required bind:value={confirmation} disabled={loading} />
			</div>
			<Error {error} class="font-inter" />
			<Dialog.Footer class="mt-4">
				<Button
					onclick={() => (open = false)}
					className="text-sm font-medium w-fit py-1.5 rounded-lg">Cancel</Button
				>
				<Button
					type="submit"
					onclick={(e) => {
						e.preventDefault();
						open = true;
						handleChangePassword();
					}}
					disabled={!(currentPassword.length > 0 && passwordsMatch && newPassword.length > 0) ||
						loading}
					className="text-sm font-medium w-fit from-blue-300 to-blue-200 hover:from-blue-200 hover:to-blue-100 rounded-lg py-1.5"
					>{#if loading}
						<Spinner />
					{:else}
						Change Password
					{/if}</Button
				>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
