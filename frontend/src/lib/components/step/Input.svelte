<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { EllipsisVertical } from '@lucide/svelte';
	import TextArea from '../textarea/TextArea.svelte';
	import type { Step } from '$lib/recipes';

	interface Props {
		step: Step;
		onInstructionChange?: () => void;
		onDelete?: () => void;
	}

	let { step = $bindable(), onInstructionChange, onDelete }: Props = $props();
</script>

<div class="flex">
	<label for="step" class="grow text-lg">Step {step.step_number}</label>
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
<TextArea
	bind:value={step.instruction}
	name="step"
	className="block w-full mt-1"
	placeholder="Enter instructions"
	oninput={onInstructionChange}
/>
