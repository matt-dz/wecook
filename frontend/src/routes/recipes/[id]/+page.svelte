<script lang="ts">
	import type { PageProps } from './$types';
	import RecipePage from '$lib/components/recipe/Page.svelte';

	let { data }: PageProps = $props();
</script>

<RecipePage recipe={data.recipe} />
