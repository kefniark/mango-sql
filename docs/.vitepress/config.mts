import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Mango SQL",
  description: "Documentation Website for mango sql (getting started, samples, playground, references, ...)",
  base: '/mango-sql/',
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Getting Started', link: '/getting-started' }
    ],

    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'Install CLI', link: '/getting-started/' },
          { text: 'Client Usage', link: '/getting-started/usage' }
        ]
      },
      {
        text: 'API Reference',
        items: [
          { text: 'Mutations', link: '/api/mutations' },
          { text: 'Queries', link: '/api/queries' },
          { text: 'Filtering & Sorting', link: '/api/filtering' },
          { text: 'Transactions', link: '/api/transactions' },
          { text: 'Custom Queries', link: '/api/custom-queries' },
        ]
      },
      {
        text: 'Advanced Features',
        items: [
          // { text: 'UUID Generation', link: '/api/mutations' },
          { text: 'ERD Diagram', link: '/features/diagram' },
          { text: 'Logging', link: '/features/logging' },
          { text: 'Soft Delete', link: '/features/soft-delete' },
          // { text: 'Migrations', link: '/api/mutations' },
          { text: 'Benchmark', link: '/bench/bench' },
        ]
      },
      // {
      //   text: 'Examples',
      //   items: [
      //     { text: 'Markdown Examples', link: '/markdown-examples' },
      //     { text: 'Runtime API Examples', link: '/api-examples' }
      //   ]
      // }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/kefniark/mango-sql' }
    ],

    lastUpdated: {
      text: 'Updated at',
      formatOptions: {
        dateStyle: 'full',
        timeStyle: 'medium'
      }
    }
  }
})
