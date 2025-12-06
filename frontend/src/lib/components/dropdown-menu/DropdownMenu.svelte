<script lang="ts">
	import * as Select from '$lib/components/ui/select/index.js';
	import { TimeUnit, type TimeUnitType } from '$lib/recipes';
	interface Props {
		value?: TimeUnitType;
	}
	let { value = $bindable() }: Props = $props();

	const capitalize = (s: string) => {
		if (s.length === 0) return s;
		return s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
	};
</script>

<Select.Root type="single" bind:value>
	<Select.Trigger class="w-24 border-gray-400 bg-gray-200/50 font-inter">
		{value ? capitalize(value) : 'Unit'}
	</Select.Trigger>
	<Select.Content class="border-gray-400 bg-gray-200">
		{#each Object.keys(TimeUnit.enum) as v (v)}
			<Select.Item class="font-inter" value={v} label={capitalize(v)} />
		{/each}
	</Select.Content>
</Select.Root>
