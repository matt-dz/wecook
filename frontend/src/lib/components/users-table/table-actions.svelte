<script lang="ts">
	import EllipsisIcon from '@lucide/svelte/icons/ellipsis';
	import { Button as ShadButton } from '$lib/components/ui/button/index.js';
	import Button from '$lib/components/button/Button.svelte';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { toast } from 'svelte-sonner';
	import { deleteUser } from '$lib/users';
	import fetch from '$lib/http';
	import { HTTPError } from 'ky';
	import { parseError } from '$lib/errors/api';

	let {
		id,
		email,
		onUserDeleted,
		isCurrentUser,
		canDelete
	}: {
		id: number;
		email: string;
		onUserDeleted: (userId: number) => void;
		isCurrentUser: boolean;
		canDelete: boolean;
	} = $props();

	let deleteDialogOpen = $state(false);
	let deleting = $state(false);

	const handleDeleteUser = async () => {
		try {
			deleting = true;
			await deleteUser(fetch, { user_id: id });
			toast.success('User deleted successfully.');
			onUserDeleted(id);
		} catch (e) {
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				if (err.success) {
					console.error('failed to delete user', err.data);
				}
			}
			console.error('failed to delete user', e);
			toast.error('Failed to delete user.');
		} finally {
			deleting = false;
			deleteDialogOpen = false;
		}
	};

	const handleCopyUserId = async () => {
		try {
			await navigator.clipboard.writeText(id.toString());
			toast.success('User ID copied.');
		} catch (e) {
			console.error('failed to write text', e);
			toast.error('Failed to copy user id.');
		}
	};
</script>

{#snippet cancel()}
	<Button className="rounded-lg text-sm" onclick={() => (deleteDialogOpen = false)}>Cancel</Button>
{/snippet}

{#snippet del()}
	<Button
		className="rounded-lg text-sm from-red-300 to-red-200 hover:from-red-200 hover:to-red-100"
		onclick={handleDeleteUser}>Delete</Button
	>
{/snippet}

<AlertDialog.Root bind:open={deleteDialogOpen}>
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<ShadButton {...props} variant="ghost" size="icon" class="relative size-8 p-0">
					<span class="sr-only">Open menu</span>
					<EllipsisIcon />
				</ShadButton>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content>
			<DropdownMenu.Group>
				<DropdownMenu.Label class="font-inter">Actions</DropdownMenu.Label>
				<DropdownMenu.Item class="font-inter" onclick={handleCopyUserId}>
					Copy user ID
				</DropdownMenu.Item>
			</DropdownMenu.Group>
			<DropdownMenu.Separator />
			{#if canDelete}
				<AlertDialog.Trigger class="w-full">
					<DropdownMenu.Item
						class="font-inter text-red-500 data-highlighted:bg-red-100 data-highlighted:text-red-500"
						>Delete User</DropdownMenu.Item
					>
				</AlertDialog.Trigger>
			{:else}
				<DropdownMenu.Item
					disabled
					class="font-inter text-gray-400"
					title="Cannot delete the only user in the system"
				>
					Delete User
				</DropdownMenu.Item>
			{/if}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
	<AlertDialog.Content>
		<AlertDialog.Title class="font-inter text-red-500">Delete User</AlertDialog.Title>
		<AlertDialog.Description class="font-inter">
			{#if isCurrentUser}
				<span class="font-bold text-red-600"
					>⚠️ Warning: You are about to delete your own account!</span
				>
				<br /><br />
				This will delete <span class="font-bold">your account</span> ({email}) and
				<span class="font-bold">all of your recipes</span>. You will be immediately logged out and
				will lose access to the admin dashboard. This action
				<span class="font-bold">CANNOT</span> be undone.
			{:else}
				This will delete the user with email <span class="font-bold">{email}</span> and all of their
				recipes. This action <span class="font-bold">CANNOT</span> be undone.
			{/if}
		</AlertDialog.Description>
		<AlertDialog.Footer>
			<AlertDialog.Cancel disabled={deleting} child={cancel} />
			<AlertDialog.Action disabled={deleting} child={del} />
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
