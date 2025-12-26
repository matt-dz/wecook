<script lang="ts">
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import Button from '../button/Button.svelte';

	interface Props {
		open?: boolean;
		onConfirmation?: () => void;
		onDenial?: () => void;
	}

	let { open = $bindable(false), onConfirmation, onDenial }: Props = $props();
</script>

{#snippet confirm()}
	<Button
		onclick={() => {
			open = false;
			onConfirmation?.();
		}}
		className="text-sm font-medium w-fit from-red-300 to-red-200 hover:from-red-200 hover:to-red-100"
		>Unpublish</Button
	>
{/snippet}

{#snippet cancel()}
	<Button
		onclick={() => {
			open = false;
			onDenial?.();
		}}
		className="text-sm font-medium w-fit">Cancel</Button
	>
{/snippet}

<AlertDialog.Root bind:open>
	<AlertDialog.Portal>
		<AlertDialog.Content>
			<AlertDialog.Header>
				<AlertDialog.Title class="text-red-500">Unpublish Recipe</AlertDialog.Title>
				<AlertDialog.Description>
					Are you sure you want to <span class="font-semibold">unpublish</span> this recipe? The
					recipe will be
					<span class="italic">unavailable</span> to the public.
				</AlertDialog.Description>
				<AlertDialog.Footer>
					<AlertDialog.Cancel child={cancel} />
					<AlertDialog.Action child={confirm} />
				</AlertDialog.Footer>
			</AlertDialog.Header>
		</AlertDialog.Content>
	</AlertDialog.Portal>
</AlertDialog.Root>
