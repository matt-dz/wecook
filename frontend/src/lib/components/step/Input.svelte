<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { EllipsisVertical } from '@lucide/svelte';
	import { Textarea } from '$lib/components/ui/textarea/index.js';

	import ImagePreview from '$lib/components/image/ImagePreview.svelte';
	import ImageInput from '$lib/components/ImageInput.svelte';
	import type { Step } from '$lib/recipes';

	interface Props {
		step: Step;
		onInstructionChange?: (stepID: number) => void;
		onDelete?: (stepID: number) => void;
		onImageUpload?: (stepID: number, image: File) => void;
		onImageDeletion?: (stepID: number) => void;
	}

	let fileInput: HTMLInputElement | undefined = $state();
	const openFilePicker = () => {
		fileInput?.click();
	};

	const handleFileSelect = (e: Event) => {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (file) {
			onImageUpload?.(step.id, file);
		}
	};

	let {
		step = $bindable(),
		onInstructionChange,
		onDelete,
		onImageUpload,
		onImageDeletion
	}: Props = $props();
</script>

<div>
	<div class="flex">
		<label for="step" class="grow text-lg">Step {step.step_number}</label>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger class="-mr-1 rounded-lg p-1 hover:bg-gray-200">
				<EllipsisVertical strokeWidth={1.5} size={20} fill="black" />
			</DropdownMenu.Trigger>
			<DropdownMenu.Content>
				<DropdownMenu.Group>
					<DropdownMenu.Item class="p-0">
						<button onclick={openFilePicker} class="w-full px-2 py-1.5 text-left">Add Image</button>
					</DropdownMenu.Item>
				</DropdownMenu.Group>
				<DropdownMenu.Separator />
				<DropdownMenu.Group>
					<DropdownMenu.Item class="p-0">
						<button
							onclick={() => onDelete?.(step.id)}
							class=" w-full px-2 py-1.5 text-left text-red-500"
						>
							Delete
						</button>
					</DropdownMenu.Item>
				</DropdownMenu.Group>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>
	<Textarea
		bind:value={step.instruction}
		name="step"
		class="mt-1 block w-full"
		placeholder="Enter instructions."
		oninput={() => onInstructionChange?.(step.id)}
	/>
	{#if step.image_url}
		<div class="pt-4">
			<ImagePreview
				src={step.image_url}
				alt={'step ' + step.step_number}
				onRemove={() => onImageDeletion?.(step.id)}
			/>
		</div>
	{/if}

	<ImageInput bind:ref={fileInput} onchange={handleFileSelect} class="hidden" />
</div>
