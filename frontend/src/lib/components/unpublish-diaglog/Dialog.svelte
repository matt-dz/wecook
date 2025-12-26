<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import Button from '../button/Button.svelte';
	interface Props {
		onConfirmation?: () => void;
		onDenial?: () => void;
	}
	let { onConfirmation, onDenial }: Props = $props();
</script>

<div class="flex">
	<Dialog.Trigger>
		<Button
			className="text-sm font-medium w-fit mt-6 from-red-300 to-red-200 hover:from-red-200 hover:to-red-100"
			>Unpublish</Button
		>
	</Dialog.Trigger>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title class="text-red-500">Unpublish Recipe</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to <span class="font-semibold">unpublish</span> this recipe? The
				recipe will be
				<span class="italic">unavailable</span> to the public.
			</Dialog.Description>
			<Dialog.Footer>
				<Dialog.Close onclick={onDenial}>
					<Button className="text-sm font-medium w-fit">Cancel</Button>
					<Button
						onclick={onConfirmation}
						className="text-sm font-medium w-fit from-red-300 to-red-200 hover:from-red-200 hover:to-red-100"
						>Unpublish</Button
					>
				</Dialog.Close>
			</Dialog.Footer>
		</Dialog.Header>
	</Dialog.Content>
</div>
