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
			className="text-sm font-medium w-fit mt-6 from-green-300 to-green-200 hover:from-green-200 hover:to-green-100"
			>Publish</Button
		>
	</Dialog.Trigger>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title class="text-green-600">Publish Recipe</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to <span class="font-semibold">Publish</span> this recipe? The recipe
				will become publicly
				<span class="italic">available</span>.
			</Dialog.Description>
			<Dialog.Footer>
				<Dialog.Close onclick={onDenial}>
					<Button className="text-sm font-medium w-fit">Cancel</Button>
					<Button
						onclick={onConfirmation}
						className="text-sm font-medium w-fit from-green-300 to-green-200 hover:from-green-200 hover:to-green-100"
						>Publish</Button
					>
				</Dialog.Close>
			</Dialog.Footer>
		</Dialog.Header>
	</Dialog.Content>
</div>
