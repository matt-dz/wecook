<script lang="ts">
	import Input from '$lib/components/input/Input.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import type { Ingredient } from '$lib/recipes';
	import { EllipsisVertical } from '@lucide/svelte';

	interface Props {
		ingredient: Ingredient;
		onQuantityChange?: () => void;
		onUnitChange?: () => void;
		onNameChange?: () => void;
		onDelete?: () => void;
	}

	let {
		ingredient = $bindable(),
		onQuantityChange,
		onUnitChange,
		onNameChange,
		onDelete
	}: Props = $props();

	const onlyPositiveNumbers = (e: KeyboardEvent) => {
		const invalid = ['e', 'E', '+', '-'];
		if (invalid.includes(e.key)) e.preventDefault();
	};
</script>

<div class="flex items-center gap-2">
	<Input
		className="w-20"
		placeholder="Quantity"
		type="number"
		onkeydown={onlyPositiveNumbers}
		bind:value={ingredient.quantity}
		oninput={onQuantityChange}
	/>
	<Input className="w-20" placeholder="Unit" bind:value={ingredient.unit} oninput={onUnitChange} />
	<p class="inline-block">of</p>
	<Input className="w-60" bind:value={ingredient.name} placeholder="Name" oninput={onNameChange} />
	<DropdownMenu.Root>
		<DropdownMenu.Trigger class="-mr-1 rounded-full p-1 hover:bg-gray-200">
			<EllipsisVertical strokeWidth={1.5} size={20} fill="black" />
		</DropdownMenu.Trigger>
		<DropdownMenu.Content>
			<DropdownMenu.Group>
				<DropdownMenu.Item>
					<p>Add Image</p>
				</DropdownMenu.Item>
			</DropdownMenu.Group>
			<DropdownMenu.Group>
				<DropdownMenu.Item>
					<button onclick={onDelete} class=" w-full text-left text-red-500"> Delete </button>
				</DropdownMenu.Item>
			</DropdownMenu.Group>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
</div>
