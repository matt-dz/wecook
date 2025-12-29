<script lang="ts">
	import type { PageProps } from './$types';
	import RecipePage from '$lib/components/recipe/Page.svelte';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>{data.recipe.recipe.title} | WeCook</title>
</svelte:head>

<RecipePage recipe={data.recipe} />
