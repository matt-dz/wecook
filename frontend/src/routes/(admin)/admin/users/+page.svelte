<script lang="ts">
	import Button from '$lib/components/button/Button.svelte';
	import DataTable from '$lib/components/users-table/Table.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { columns } from '$lib/components/users-table/columns';
	import type { PageProps } from './$types';
	import InviteUserDialog from '$lib/components/invite-user-dialog/Dialog.svelte';

	let { data }: PageProps = $props();
	const users = $state(data.users);
</script>

<Dialog.Root>
	<div class="mt-12 flex justify-center">
		<div class="w-full max-w-3xl">
			<Dialog.Trigger>
				<Button className="mb-4 from-blue-300 to-blue-200 hover:from-blue-200 hover:to-blue-100"
					>Invite User</Button
				>
			</Dialog.Trigger>
			<DataTable data={users.users} {columns} />
		</div>
	</div>
	<InviteUserDialog />
</Dialog.Root>
