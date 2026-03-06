// @ts-check

import {themes as prismThemes} from 'prism-react-renderer';

const siteUrl = 'https://emmesbef.github.io';
const pagesBaseUrl = '/clockify-jira-sync/';
const coverageUrl = `${siteUrl}${pagesBaseUrl}coverage/`;

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Clockify ↔ Jira Time Sync Docs',
  tagline: 'Project documentation for setup, development, testing, and releases.',
  favicon: 'img/favicon.ico',
  future: {
    v4: true,
  },
  url: siteUrl,
  baseUrl: pagesBaseUrl,
  organizationName: 'emmesbef',
  projectName: 'clockify-jira-sync',
  onBrokenLinks: 'throw',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },
  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: '/',
          sidebarPath: './sidebars.js',
          editUrl: 'https://github.com/emmesbef/clockify-jira-sync/tree/main/docs-site/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],
  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      colorMode: {
        respectPrefersColorScheme: true,
      },
      navbar: {
        title: 'Clockify ↔ Jira Sync',
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docsSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            href: coverageUrl,
            label: 'Coverage',
            position: 'left',
          },
          {
            href: 'https://github.com/emmesbef/clockify-jira-sync',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Overview',
                to: '/',
              },
              {
                label: 'Setup & configuration',
                to: '/setup-configuration',
              },
              {
                label: 'Development, build, and test',
                to: '/development-build-test',
              },
              {
                label: 'Releases & CI/CD',
                to: '/release-cicd',
              },
            ],
          },
          {
            title: 'Project',
            items: [
              {
                label: 'Repository',
                href: 'https://github.com/emmesbef/clockify-jira-sync',
              },
              {
                label: 'Issues',
                href: 'https://github.com/emmesbef/clockify-jira-sync/issues',
              },
              {
                label: 'GitHub Releases',
                href: 'https://github.com/emmesbef/clockify-jira-sync/releases',
              },
            ],
          },
          {
            title: 'Automation',
            items: [
              {
                label: 'CI workflow',
                href: 'https://github.com/emmesbef/clockify-jira-sync/actions/workflows/ci.yml',
              },
              {
                label: 'Coverage dashboard',
                href: coverageUrl,
              },
              {
                label: 'Release workflow',
                href: 'https://github.com/emmesbef/clockify-jira-sync/actions/workflows/release.yml',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Clockify ↔ Jira Time Sync. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
      },
    }),
};

export default config;
