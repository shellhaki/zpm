// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  integrations: [
    starlight({
      title: 'ZPM',
      description: 'A small, sharp process manager for apps you want to keep alive.',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/shellhaki/zpm' },
        { icon: 'x.com', label: 'X', href: 'https://x.com/haki_xer' },
      ],
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        {
          label: 'Getting started',
          items: [
            { label: 'Overview', slug: 'index' },
          ],
        },
      ],
    }),
  ],
});