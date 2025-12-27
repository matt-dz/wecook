<script lang="ts">
	import * as Select from '$lib/components/ui/select/index.js';
	import { TimeUnit, type TimeUnitType } from '$lib/recipes';
	interface Props {
		value?: TimeUnitType;
		onValueChange?: (val: string) => void;
	}
	let { value = $bindable(), onValueChange }: Props = $props();

	const capitalize = (s: string) => {
		if (s.length === 0) return s;
		return s.charAt(0).toUpperCase() + s.slice(1).toLowerCase();
	};
</script>

<Select.Root type="single" bind:value {onValueChange}>
	<Select.Trigger>
		{value ? capitalize(value) : 'Unit'}
	</Select.Trigger>
	<Select.Content>
		{#each Object.keys(TimeUnit.enum) as v (v)}
			<Select.Item value={v} label={capitalize(v)} />
		{/each}
	</Select.Content>
</Select.Root>
