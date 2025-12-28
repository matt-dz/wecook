<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { updatePersonalRecipe } from '$lib/recipes';
	import { formatDuration, toMinutes } from '$lib/time';
	import fetch from '$lib/http';
	import { resolve } from '$app/paths';
	import { HTTPError } from 'ky';
	import { toast } from 'svelte-sonner';
	import { twMerge } from 'tailwind-merge';
	import { EllipsisVertical } from '@lucide/svelte';
	import PublishDialog from '$lib/components/publish-dialog/Dialog.svelte';
	import UnpublishDialog from '$lib/components/unpublish-diaglog/Dialog.svelte';
	import ShareDialog from '$lib/components/share-dialog/Dialog.svelte';
	import clsx from 'clsx';
	import type { RecipeAndOwner } from '$lib/recipes';
	import { parseError } from '$lib/errors/api';
	interface Props {
		recipe: RecipeAndOwner;
		className?: string;
		editable?: boolean;
		personal?: boolean;
	}

	let { recipe, className, editable = false, personal = false }: Props = $props();

	let published = $state(recipe.recipe.published);
	let publishDialogOpen = $state(false);
	let unpublishDialogOpen = $state(false);
	let shareDialogOpen = $state(false);

	const totalCookTime =
		(recipe.recipe?.cook_time_amount && recipe.recipe?.cook_time_unit
			? toMinutes(recipe.recipe.cook_time_amount, recipe.recipe.cook_time_unit)
			: 0) +
		(recipe.recipe?.prep_time_amount && recipe.recipe?.prep_time_unit
			? toMinutes(recipe.recipe.prep_time_amount, recipe.recipe.prep_time_unit)
			: 0);
	const title =
		recipe.recipe.title.trim().length > 0 ? recipe.recipe.title.trim() : 'Untitled Recipe';

	const togglePublish = async () => {
		try {
			await updatePersonalRecipe(fetch, {
				recipe_id: recipe.recipe.id,
				published: !published
			});
			published = !published;
			toast.success(`Recipe ${published ? '' : 'un'}published successfully.`);
		} catch (e) {
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				if (err.success) {
					console.error('failed to publish recipe', err.data);
				}
			}
			console.error(e);
			toast.error('Failed to publish recipe.');
		}
	};
</script>

<a
	class="h-fit w-fit rounded-3xl border-solid border-gray-400/50 p-3 shadow-none transition-shadow duration-250 hover:shadow-[0_0_12px_rgba(0,0,0,0.5)]"
	href={resolve(personal ? `/recipes/${recipe.recipe.id}/preview` : `/recipes/${recipe.recipe.id}`)}
>
	{#if recipe.recipe.image_url !== undefined}
		<div
			class={twMerge(
				clsx(
					'h-40 w-[260px] overflow-hidden rounded-xl border-2 border-solid border-black',
					className
				)
			)}
		>
			<img
				src={recipe.recipe.image_url}
				alt="cover"
				class="h-full w-full object-cover object-center"
			/>
		</div>
	{:else}
		<div
			class={twMerge(
				clsx('h-40 w-[260px] rounded-xl border border-solid border-black bg-blue-200', className)
			)}
		></div>
	{/if}

	<div class="mt-1 flex items-start">
		<div class="flex-1">
			<h1 class="text-lg font-semibold">{title}</h1>
			<h2 class="text-sm capitalize">{recipe.owner.first_name} {recipe.owner.last_name}</h2>
			<h3 class="text-sm text-gray-400">{formatDuration(totalCookTime, 'minutes')}</h3>
		</div>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<button {...props} class="rounded-lg p-1 hover:bg-gray-200">
						<EllipsisVertical />
					</button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content>
				<DropdownMenu.Group>
					{#if editable}
						<DropdownMenu.Item>
							<a class="w-full" href={resolve(`/recipes/${recipe.recipe.id}/edit`)}> Edit </a>
						</DropdownMenu.Item>
						{#if !published}
							<DropdownMenu.Item
								onSelect={(e) => {
									e.preventDefault();
									publishDialogOpen = true;
								}}>Publish</DropdownMenu.Item
							>
						{:else}
							<DropdownMenu.Item
								onSelect={(e) => {
									e.preventDefault();
									unpublishDialogOpen = true;
								}}>Unpublish</DropdownMenu.Item
							>
						{/if}
					{/if}
				</DropdownMenu.Group>
				{#if published}
					<DropdownMenu.Group>
						<DropdownMenu.Item
							onSelect={(e) => {
								e.preventDefault();
								shareDialogOpen = true;
							}}>Share</DropdownMenu.Item
						>
					</DropdownMenu.Group>
				{/if}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
		<PublishDialog bind:open={publishDialogOpen} onConfirmation={togglePublish} />
		<UnpublishDialog bind:open={unpublishDialogOpen} onConfirmation={togglePublish} />
		<ShareDialog bind:open={shareDialogOpen} recipeId={recipe.recipe.id} />
	</div>
</a>
