<script lang="ts">
  import { Languages, Check, ChevronDown } from 'lucide-svelte';
  import { i18n, SUPPORTED_LOCALES_LIST, type SupportedLocale } from './i18n.svelte';
  import * as m from '$lib/paraglide/messages.js';

  interface Props {
    onLocaleChange?: (locale: SupportedLocale) => void;
  }

  let { onLocaleChange }: Props = $props();

  let isOpen = $state<boolean>(false);
  let dropdownRef: HTMLDivElement | null = null;

  function selectLocale(code: SupportedLocale) {
    i18n.set(code);
    isOpen = false;
    if (onLocaleChange) {
      onLocaleChange(code);
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      isOpen = false;
    }
  }

  // Click outside to close
  function handleWindowClick(e: MouseEvent) {
    if (isOpen && dropdownRef && !dropdownRef.contains(e.target as Node)) {
      isOpen = false;
    }
  }
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleKeydown} />

<div class="relative inline-block text-left" bind:this={dropdownRef}>
  <button
    type="button"
    onclick={() => (isOpen = !isOpen)}
    class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-neutral-900/90 border border-neutral-800 hover:border-neutral-700 text-xs font-medium text-neutral-300 hover:text-white transition shadow-sm cursor-pointer"
    title={m.settings_language()}
    aria-haspopup="true"
    aria-expanded={isOpen}
  >
    <Languages class="w-3.5 h-3.5 text-[#E50914] shrink-0" />
    <span class="font-mono font-bold tracking-wider uppercase text-neutral-200">
      {i18n.current.toUpperCase()}
    </span>
    <span class="hidden md:inline text-neutral-400 text-[11px]">
      {i18n.info.nativeLabel}
    </span>
    <ChevronDown class="w-3 h-3 text-neutral-500 transition-transform duration-150 {isOpen ? 'rotate-180' : ''}" />
  </button>

  {#if isOpen}
    <div
      class="absolute right-0 mt-2 w-56 rounded-xl bg-neutral-900/95 backdrop-blur-md border border-neutral-800 shadow-2xl p-1.5 z-50 animate-in fade-in zoom-in-95 duration-100"
      role="menu"
    >
      <div class="px-2 py-1 mb-1 border-b border-neutral-800/80 text-[10px] uppercase font-bold tracking-wider text-neutral-500">
        {m.settings_language()}
      </div>

      {#each SUPPORTED_LOCALES_LIST as loc}
        {@const isSelected = i18n.current === loc.code}
        <button
          type="button"
          onclick={() => selectLocale(loc.code)}
          class="flex items-center justify-between w-full px-2.5 py-2 text-xs rounded-lg text-left transition {isSelected
            ? 'bg-[#E50914]/15 text-[#E50914] font-semibold'
            : 'text-neutral-300 hover:bg-neutral-800/90 hover:text-white'}"
          role="menuitem"
        >
          <div class="flex items-center gap-2.5 truncate">
            <span
              class="px-1.5 py-0.5 rounded text-[10px] font-mono font-bold tracking-wider {isSelected
                ? 'bg-[#E50914] text-white'
                : 'bg-neutral-800 text-neutral-400'}"
            >
              {loc.code.toUpperCase()}
            </span>
            <div class="flex flex-col truncate">
              <span class="truncate leading-tight text-neutral-200">{loc.nativeLabel}</span>
              <span class="text-[10px] text-neutral-500 truncate">{loc.country}</span>
            </div>
          </div>

          {#if isSelected}
            <Check class="w-3.5 h-3.5 text-[#E50914] shrink-0 ms-2" />
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>
