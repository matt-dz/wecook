<script lang="ts">
	import type { PageProps } from './$types';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import Button from '$lib/components/button/Button.svelte';
	import ChangePasswordDialog from '$lib/components/change-password-dialog/Dialog.svelte';
	import { LockKeyhole } from '@lucide/svelte';

	let { data }: PageProps = $props();
	let changePasswordDialogOpen = $state(false);

	const user = data.user;
</script>

<svelte:head>
	<title>Profile | WeCook</title>
</svelte:head>

<div class="absolute top-0 bottom-0 flex w-full items-center justify-center px-4">
	<div
		class="flex w-full max-w-[375px] flex-col gap-4 rounded-lg border border-solid p-6 shadow-xs"
	>
		<div class="space-y-2">
			<Label class="font-inter">First Name</Label>
			<Input disabled class="font-inter" value={user.first_name} />
		</div>
		<div class="space-y-2">
			<Label class="font-inter">Last Name</Label>
			<Input disabled class="font-inter" value={user.last_name} />
		</div>
		<div class="space-y-2">
			<Label class="font-inter">Email</Label>
			<Input disabled class="font-inter" value={user.email} />
		</div>
		<div class="space-y-2">
			<Label class="font-inter">Role</Label>
			<Input disabled class="font-inter" value={user.role} />
		</div>
		<div class="space-y-2">
			<Label class="font-inter">Password</Label>
			<Button
				onclick={(e) => {
					e.preventDefault();
					changePasswordDialogOpen = true;
				}}
				className="text-sm w-fit rounded-lg flex justify-center py-1.5 gap-2"
			>
				<LockKeyhole size={18} strokeWidth={1.5} />
				Change Password</Button
			>
		</div>
	</div>
</div>

<ChangePasswordDialog bind:open={changePasswordDialogOpen} />
