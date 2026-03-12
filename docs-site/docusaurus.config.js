// @ts-check

import {themes as prismThemes} from 'prism-react-renderer';

const siteUrl = 'https://level-87.gitlab.io';
const baseUrl = '/';
const coverageUrl = `${siteUrl}${baseUrl}coverage/`;

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'JiraFy Clockwork Docs',
  tagline: 'Project documentation for setup, development, testing, and releases.',
  favicon: 'img/favicon.ico',
  future: {
    v4: true,
  },
  url: siteUrl,
  baseUrl,
  organizationName: 'level-87',
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
          editUrl: 'https://gitlab.com/level-87/clockify-jira-sync/-/tree/main/docs-site/',
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
        title: 'JiraFy Clockwork',
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
            href: 'https://gitlab.com/level-87/clockify-jira-sync',
            label: 'GitLab',
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
                href: 'https://gitlab.com/level-87/clockify-jira-sync',
              },
              {
                label: 'Issues',
                href: 'https://gitlab.com/level-87/clockify-jira-sync/-/issues',
              },
              {
                label: 'GitLab Releases',
                href: 'https://gitlab.com/level-87/clockify-jira-sync/-/releases',
              },
            ],
          },
          {
            title: 'Automation',
            items: [
              {
                label: 'CI pipelines',
                href: 'https://gitlab.com/level-87/clockify-jira-sync/-/pipelines',
              },
              {
                label: 'Coverage dashboard',
                href: coverageUrl,
              },
              {
                label: 'Release tags',
                href: 'https://gitlab.com/level-87/clockify-jira-sync/-/tags',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} JiraFy Clockwork. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
      },
    }),
};

export default config;
