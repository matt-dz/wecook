<script lang="ts">
	import { Input } from '$lib/components/ui/input/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import ImagePreview from '$lib/components/image/ImagePreview.svelte';
	import { type Ingredient } from '$lib/recipes';
	import { EllipsisVertical } from '@lucide/svelte';

	interface Props {
		ingredient: Ingredient;
		onQuantityChange?: (ingredientID: number) => void;
		onUnitChange?: (ingredientID: number) => void;
		onNameChange?: (ingredientID: number) => void;
		onDelete?: (ingredientID: number) => void;
		onImageUpload?: (ingredientID: number, image: File) => void;
		onImageDeletion?: (ingredientID: number) => void;
	}

	let {
		ingredient = $bindable(),
		onQuantityChange,
		onUnitChange,
		onNameChange,
		onDelete,
		onImageUpload,
		onImageDeletion
	}: Props = $props();

	let fileInput: HTMLInputElement;

	const onlyPositiveNumbers = (e: KeyboardEvent) => {
		const invalid = ['e', 'E', '+', '-'];
		if (invalid.includes(e.key)) e.preventDefault();
	};

	const handleFileSelect = (e: Event) => {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (file) {
			onImageUpload?.(ingredient.id, file);
		}
	};

	const removeImage = () => {
		onImageDeletion?.(ingredient.id);
	};

	const openFilePicker = () => {
		fileInput?.click();
	};
</script>

<div class="space-y-2">
	<div class="flex items-center gap-2">
		<Input
			class="w-20"
			placeholder="Quantity"
			type="number"
			onkeydown={onlyPositiveNumbers}
			bind:value={ingredient.quantity}
			oninput={() => onQuantityChange?.(ingredient.id)}
		/>
		<Input
			class="w-20"
			placeholder="Unit"
			bind:value={ingredient.unit}
			oninput={() => onUnitChange?.(ingredient.id)}
		/>
		<p class="inline-block">of</p>
		<Input
			class="w-60"
			bind:value={ingredient.name}
			placeholder="Name"
			oninput={() => onNameChange?.(ingredient.id)}
		/>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger class="-mr-1 rounded-lg p-1 hover:bg-gray-200">
				<EllipsisVertical strokeWidth={1.5} size={20} fill="black" />
			</DropdownMenu.Trigger>
			<DropdownMenu.Content>
				<DropdownMenu.Group>
					<DropdownMenu.Item class="p-0">
						<button onclick={openFilePicker} class="h-full w-full px-2 py-1.5 text-left"
							>Add Image</button
						>
					</DropdownMenu.Item>
				</DropdownMenu.Group>
				<DropdownMenu.Separator />
				<DropdownMenu.Group>
					<DropdownMenu.Item class="p-0">
						<button
							onclick={() => onDelete?.(ingredient.id)}
							class="w-full px-2 py-1.5 text-left text-red-500">Delete</button
						>
					</DropdownMenu.Item>
				</DropdownMenu.Group>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>

	{#if ingredient.image_url}
		<div class="pt-1">
			<ImagePreview
				src={ingredient.image_url}
				alt={ingredient.name || 'Ingredient'}
				onRemove={removeImage}
			/>
		</div>
	{/if}

	<input
		type="file"
		accept="image/*"
		bind:this={fileInput}
		onchange={handleFileSelect}
		class="hidden"
	/>
</div>
