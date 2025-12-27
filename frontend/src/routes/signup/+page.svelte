<script lang="ts">
	import type { PageProps } from './$types';
	import Button from '$lib/components/button/Button.svelte';
	import Input from '$lib/components/input/Input.svelte';
	import fetch from '$lib/http';
	import { parseError } from '$lib/errors/api';
	import { goto, invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { HTTPError } from 'ky';
	import { signupRequest } from '$lib/users';
	import { toast } from 'svelte-sonner';

	const placeholderError = 'placeholder';
	let loading = $state(false);
	let firstName: string = $state('');
	let lastName: string = $state('');
	let error: string = $state(placeholderError);
	let email: string = $state('');
	let password: string = $state('');

	let { data }: PageProps = $props();

	const onSubmit = async (e: SubmitEvent) => {
		e.preventDefault();
		error = placeholderError;
		try {
			loading = true;
			await signupRequest(fetch, {
				email,
				password,
				first_name: firstName,
				last_name: lastName,
				invite_code: data.code
			});
			await invalidateAll();
			goto(resolve('/home'));
		} catch (e) {
			if (e instanceof HTTPError) {
				const err = await parseError(e.response);
				console.error(err.data);
				if (err.success) {
					error = err.data.message;
					return;
				}
			}
			console.error('failed to signup: ', e);
			toast.error('Failed to signup.');
		} finally {
			loading = false;
		}
	};
</script>

<div class="absolute top-0 bottom-0 flex w-full items-center justify-center px-4">
	<form
		class="flex w-full max-w-[375px] flex-col gap-2 rounded-2xl border border-solid p-6 shadow-lg"
		onsubmit={onSubmit}
	>
		<div class="mb-2">
			<h1 class="text-left text-lg">Sign up</h1>
			<p class="text-sm text-gray-500">Welcome to WeCook!</p>
		</div>
		<div class="flex w-full flex-col space-y-1">
			<label for="fname" class="text-sm">First Name</label>
			<Input
				disabled={loading}
				className="text-sm"
				bind:value={firstName}
				id="fname"
				name="fname"
				placeholder="first name"
				autocomplete="given-name"
				autocorrect="off"
				spellcheck="false"
				required
			/>
		</div>

		<div class="flex w-full flex-col space-y-1">
			<label for="lname" class="text-sm">Last Name</label>
			<Input
				disabled={loading}
				className="text-sm"
				bind:value={lastName}
				id="lname"
				name="lname"
				placeholder="last name"
				autocomplete="family-name"
				autocorrect="off"
				spellcheck="false"
				required
			/>
		</div>

		<div class="flex w-full flex-col space-y-1">
			<label for="email" class="text-sm">Email</label>
			<Input
				disabled={loading}
				className="text-sm"
				bind:value={email}
				id="email"
				name="email"
				type="email"
				placeholder="email"
				autocomplete="email"
				autocapitalize="none"
				autocorrect="off"
				spellcheck="false"
				inputmode="email"
				required
			/>
		</div>

		<div class="mt-1 flex w-full flex-col space-y-1">
			<label for="email" class="text-sm">Password</label>
			<Input
				disabled={loading}
				className="text-sm"
				bind:value={password}
				id="password"
				name="password"
				type="password"
				placeholder="password"
				autocomplete="current-password"
				autocapitalize="none"
				autocorrect="off"
				spellcheck="false"
				inputmode="text"
				required
			/>
		</div>

		<p
			class="text-center text-sm text-red-600"
			class:invisible={error === placeholderError}
			class:visible={error !== placeholderError}
		>
			{error}
		</p>

		<Button
			className="from-blue-300 to-blue-200 border-blue-400 hover:from-blue-200 hover:to-blue-100"
			type="submit"
			disabled={loading}
		>
			<p class="font-medium">Sign up</p>
		</Button>
	</form>
</div>
