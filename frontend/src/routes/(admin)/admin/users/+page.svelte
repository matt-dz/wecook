<script lang="ts">
	import Button from '$lib/components/button/Button.svelte';
	import DataTable from '$lib/components/users-table/Table.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { getColumns } from '$lib/components/users-table/columns';
	import type { PageProps } from './$types';
	import InviteUserDialog from '$lib/components/invite-user-dialog/Dialog.svelte';

	let { data }: PageProps = $props();
	let users = $state(data.users.users);

	let inviteDialogOpen = $state(false);
	let inviteEmail = $state('');

	const handleUserDeleted = (userId: number) => {
		users = users.filter((user) => user.id !== userId);
	};

	const columns = getColumns(handleUserDeleted);

	$effect(() => {
		if (!inviteDialogOpen) {
			inviteEmail = '';
		}
	});
</script>

<Dialog.Root bind:open={inviteDialogOpen}>
	<div class="mt-12 flex justify-center">
		<div class="w-full max-w-3xl px-6">
			<Dialog.Trigger>
				<Button
					className="mb-4 from-blue-300 to-blue-200 hover:from-blue-200 hover:to-blue-100 rounded-lg text-sm"
					>Invite User</Button
				>
			</Dialog.Trigger>
			<DataTable data={users} {columns} />
		</div>
	</div>
	<InviteUserDialog bind:email={inviteEmail} />
</Dialog.Root>
