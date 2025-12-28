<script lang="ts">
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
</script>

<div class="mt-12 flex justify-center px-6">
	<div class="flex w-full max-w-3xl justify-center">
		<div class="flex items-start gap-2">
			<Switch id="public-signup" disabled />
			<div class="space-y-1">
				<Label for="public-signup" class="font-inter">Public Signup</Label>
				<p class="font-inter text-sm font-light text-gray-400">
					If disabled (default), users must receive an invitation to signup.
				</p>
			</div>
		</div>
	</div>
</div>
