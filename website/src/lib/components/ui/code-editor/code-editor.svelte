<script lang="ts">
	import { EditorView, basicSetup } from 'codemirror';
	import { EditorState } from '@codemirror/state';
	import { yaml } from '@codemirror/lang-yaml';
	import { StreamLanguage } from '@codemirror/language';
	import { properties } from '@codemirror/legacy-modes/mode/properties';
	import { oneDark } from '@codemirror/theme-one-dark';
	import { onMount } from 'svelte';

	interface Props {
		value: string;
		placeholder?: string;
		readonly?: boolean;
		class?: string;
		language?: 'yaml' | 'env';
	}

	let {
		value = $bindable(),
		placeholder = '',
		readonly = false,
		class: className = '',
		language = 'yaml'
	}: Props = $props();

	let editorElement: HTMLDivElement;
	let editorView: EditorView;

	onMount(() => {
		const langExtension = language === 'env' ? StreamLanguage.define(properties) : yaml();
		
		const startState = EditorState.create({
			doc: value,
			extensions: [
				basicSetup,
				langExtension,
				oneDark,
				EditorView.updateListener.of((update) => {
					if (update.docChanged) {
						value = update.state.doc.toString();
					}
				}),
				EditorState.readOnly.of(readonly),
				...(placeholder
					? [
							EditorView.theme({
								'.cm-placeholder': {
									color: '#6b7280'
								}
							})
						]
					: [])
			]
		});

		editorView = new EditorView({
			state: startState,
			parent: editorElement
		});

		return () => {
			editorView?.destroy();
		};
	});

	$effect(() => {
		if (editorView && value !== editorView.state.doc.toString()) {
			editorView.dispatch({
				changes: {
					from: 0,
					to: editorView.state.doc.length,
					insert: value
				}
			});
		}
	});
</script>

<div bind:this={editorElement} class="overflow-hidden rounded-md border {className}"></div>

<style>
	:global(.cm-editor) {
		height: 100%;
		min-height: 400px;
	}

	:global(.cm-focused) {
		outline: none;
	}
</style>
