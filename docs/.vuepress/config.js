module.exports = {
  theme: "cosmos",
  title: "Kava Documentation",
  themeConfig: {
    logo: {
      src: "/logo.svg"
    },
    custom: true,
    sidebar: {
      auto: false,
      nav: [
        {
          title: "Getting Started - Developers",
          children: [
            {
              title: "Get Started",
              path: "/Education/getting_started.html"
            },
            {
              title: "Building Modules",
              path: "/Education/hello_kava.html"
            },
            {
              title: "Javascript SDK",
              path: "/building/javascript-sdk.html"
            },
            {
              title: "Run Validator Node",
              path: "/validator_guide"
            },
            {
              title: "Run Bots",
              children: [
                {
                  title: "Sentinel Bot",
                  path: "/tools/sentinel.html"
                },
                {
                  title: "Auction Bot",
                  path: "/tools/auction.html"
                }
              ]
            },
            {
              title: "Migration Guide",
              path: "/kava-8",
              directory: true
            }
          ]
        },
        {
          title: "Resources",
          children: [
            {
              title: "Community Tools",
              path: "/tools/community.html"
            },
            {
              title: "Module Specs",
              path: "/Modules",
              directory: true,
            },
            {
              title: "CLI Docs",
              path: "/Education/user_actions/",
              directory: true,
            },
            {
              title: "REST API Spec",
              path: "https://swagger.kava.io/"
            },
            {
              title: "Protocol Reference",
              path: "https://pkg.go.dev/github.com/kava-labs/kava?tab=subdirectories"
            },
            {
              title: "FAQ", 
              path: "/Education/FAQ/",
              directory: true,
            }
          ]
        }
      ]
    },
    footer: {
      logo: "/logo.svg",
      textLink: {
        text: "kava.io",
        url: "https://www.kava.io"
      },
      services: [
        {
          service: "twitter",
          url: "https://twitter.com/kava_platform"
        },
        {
          service: "medium",
          url: "https://medium.com/kava-labs"
        },
        {
          service: "telegram",
          url: "https://t.me/kavalabs"
        },
        {
          service: "discord",
          url: "https://discord.gg/JJYnuCx"
        },
        {
          service: "github",
          url: "https://github.com/kava-labs"
        }
      ],
      smallprint: "© 2021 Kava Labs.",
      links: [
        {
          title: "Community",
          children: [
            {
              title: "Blog",
              url: "https://medium.com/kava-labs"
            },
            {
              title: "Chat",
              url: "https://t.me/kavalabs"
            }
          ]
        },
        {
          title: "Contributing",
          children: [
            {
              title: "Contributing to the docs",
              url: "https://github.com/Kava-Labs/kava/tree/master/docs"
            },
            {
              title: "Source code on GitHub",
              url: "https://github.com/Kava-Labs/kava"
            }
          ]
        },
        {
          title: "Related Docs",
          children: [
            {
              title: "Cosmos SDK",
              url: "https://cosmos.network/docs"
            },
            {
              title: "Binance Chain",
              url: "https://docs.binance.org"
            }
          ]
        }
      ]
    }
  }
}
