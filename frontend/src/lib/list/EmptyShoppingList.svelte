<script lang="ts">
  const id = $props.id();
</script>

<svg
  class="illustration"
  xmlns="http://www.w3.org/2000/svg"
  width="320"
  height="320"
  viewBox="0 0 320 320"
  aria-hidden="true"
>
  <!-- Apple adapted from Font Awesome Free 7.3.1, Copyright 2026 Fonticons, Inc. CC BY 4.0: https://fontawesome.com/license/free -->
  <defs>
    <linearGradient id={`${id}-basket`} x2="0.6" y2="1">
      <stop stop-color="var(--wa-color-neutral-50)" />
      <stop offset="1" stop-color="var(--wa-color-neutral-40)" />
    </linearGradient>
    <linearGradient id={`${id}-carton`} x2="0.35" y2="1">
      <stop stop-color="var(--wa-color-brand-80)" />
      <stop offset="1" stop-color="var(--wa-color-brand-70)" />
    </linearGradient>
    <mask id={`${id}-slots`} maskUnits="userSpaceOnUse" x="0" y="0" width="320" height="320">
      <!-- Mask luminance controls cutouts; these are not visible artwork colors. -->
      <rect width="320" height="320" fill="white" />
      <rect
        x="107.5"
        y="190"
        width="13"
        height="47"
        rx="6.5"
        fill="black"
        transform="rotate(-6 114 213.5)"
      />
      <rect x="153.5" y="190" width="13" height="47" rx="6.5" fill="black" />
      <rect
        x="199.5"
        y="190"
        width="13"
        height="47"
        rx="6.5"
        fill="black"
        transform="rotate(6 206 213.5)"
      />
    </mask>
  </defs>
  <circle
    cx="160"
    cy="160"
    r="140"
    fill="light-dark(var(--wa-color-neutral-90), var(--wa-color-neutral-fill-quiet))"
  />
  <g transform="translate(8 15) scale(0.92) rotate(-8 111 117)">
    <path
      d="M111 102c1-8-3-13-9-17"
      fill="none"
      stroke="var(--wa-color-neutral-40)"
      stroke-width="4"
      stroke-linecap="round"
    />
    <svg x="78" y="77" width="66" height="80" viewBox="0 0 448 512">
      <path
        fill={`url(#${id}-basket)`}
        d="M224 112c-8.8 0-16-7.2-16-16l0-16c0-44.2 35.8-80 80-80l16 0c8.8 0 16 7.2 16 16l0 16c0 44.2-35.8 80-80 80l-16 0zM0 288c0-76.3 35.7-160 112-160 27.3 0 59.7 10.3 82.7 19.3 18.8 7.3 39.9 7.3 58.7 0 22.9-8.9 55.4-19.3 82.7-19.3 76.3 0 112 83.7 112 160 0 128-80 224-160 224-16.5 0-38.1-6.6-51.5-11.3-8.1-2.8-16.9-2.8-25 0-13.4 4.7-35 11.3-51.5 11.3-80 0-160-96-160-224z"
      />
    </svg>
  </g>
  <g transform="translate(178 67) rotate(12 29 49) scale(0.86)">
    <rect x="11" width="46" height="10" rx="3" fill="var(--wa-color-brand-70)" />
    <path d="M11 10h46l11 22H0z" fill="var(--wa-color-brand-90)" />
    <path d="M57 10l11 22H52z" fill="var(--wa-color-brand-70)" />
    <path d="M0 32h68v78a4 4 0 0 1-4 4H4a4 4 0 0 1-4-4z" fill={`url(#${id}-carton)`} />
    <path d="M52 32h16v78a4 4 0 0 1-4 4H52z" fill="var(--wa-color-brand-70)" />
    <rect x="12" y="56" width="31" height="26" rx="3" fill="var(--wa-color-neutral-95)" />
  </g>
  <!-- Three equal rays on a regular fan, with a shared origin beside the carton. -->
  <g fill="none" stroke="var(--wa-color-brand-50)" stroke-width="4" stroke-linecap="round">
    <path d="M248.78 110.24L250.86 98.43" />
    <path d="M258.26 115.72L267.45 108.00" />
    <path d="M262.00 126.00L274.00 126.00" />
  </g>
  <path
    d="M73 170h174l-13 68q-3 15-18 15H104q-15 0-18-15z"
    fill={`url(#${id}-basket)`}
    mask={`url(#${id}-slots)`}
  />
  <path
    d="M73 170h174l-13 68q-3 15-18 15H104q-15 0-18-15z"
    fill="none"
    stroke="var(--wa-color-neutral-95)"
    stroke-opacity="0.65"
    stroke-width="1.25"
  />
  <rect
    x="60"
    y="154"
    width="200"
    height="23"
    rx="11.5"
    fill={`url(#${id}-basket)`}
    stroke="var(--wa-color-neutral-95)"
    stroke-opacity="0.65"
    stroke-width="1.25"
  />
</svg>

<style>
  .illustration {
    display: block;
    width: 12rem;
    max-width: 100%;
    height: auto;
    margin: 0 auto var(--wa-space-l);
  }
</style>
