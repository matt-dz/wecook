<script lang="ts">
	import * as AlertDialog from '$lib/components/ui/dialog/index.js';
	import { toast } from 'svelte-sonner';

	interface Props {
		open?: boolean;
		recipeId: number;
	}

	const url = () => {
		return `${window.location.origin}/recipes/${recipeId}`;
	};

	const copyUrl = async () => {
		try {
			await navigator.clipboard.writeText(url());
			toast.success('Successfully copied URL.');
		} catch (e) {
			console.error('failed to copy url', e);
			toast.error('Failed to copy URL');
		}
	};

	let { open = $bindable(false), recipeId }: Props = $props();
</script>

<AlertDialog.Root bind:open>
	<AlertDialog.Portal>
		<AlertDialog.Content>
			<AlertDialog.Header>
				<AlertDialog.Title class="text-blue-600">Share Recipe</AlertDialog.Title>
				<AlertDialog.Description>
					Share this recipe so others can view and cook it.
				</AlertDialog.Description>
			</AlertDialog.Header>
			<div class="flex w-full items-center overflow-hidden rounded-lg">
				<button
					class="shrink-0 cursor-pointer rounded-l-lg border border-r-0 border-solid border-gray-500 bg-blue-300 px-2 py-1 hover:bg-blue-200"
					onclick={copyUrl}>copy</button
				>
				<div class="min-w-0 flex-1 overflow-hidden rounded-r-lg border border-solid border-gray-500 bg-gray-100 px-2 py-1">
					<p class="overflow-hidden text-ellipsis whitespace-nowrap">{url()}</p>
				</div>
			</div>
		</AlertDialog.Content>
	</AlertDialog.Portal>
</AlertDialog.Root>
