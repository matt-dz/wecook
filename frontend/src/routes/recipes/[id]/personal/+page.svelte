<script lang="ts">
	import type { PageProps } from './$types';
	import RecipePage from '$lib/components/recipe/Page.svelte';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Recipe | {data.recipe.recipe.title}</title>
</svelte:head>

<RecipePage recipe={data.recipe} />
