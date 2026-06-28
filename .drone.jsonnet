local name = 'navidrome';
local version = '0.62.0';
local go = '1.25';
local nginx = '1.24.0';
local python = '3.12-slim-bookworm';
local platform = '26.06.01';
local playwright = 'v1.59.1-jammy';
local store_publisher = 'stable-303';
local distros = ['bookworm', 'buster'];

local platform_image(distro, arch) =
  'syncloud/platform-' + distro + '-' + arch + ':' + platform;

local build(arch, ui) = [{
  kind: 'pipeline',
  type: 'docker',
  name: arch,
  platform: {
    os: 'linux',
    arch: arch,
  },
  steps: [
    {
      name: 'nginx',
      image: 'nginx:' + nginx,
      commands: ['./nginx/build.sh'],
    },
  ] + [
    {
      name: 'nginx test ' + distro,
      image: platform_image(distro, arch),
      commands: ['./nginx/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'navidrome',
      image: 'debian:bookworm-slim',
      commands: ['./navidrome/build.sh ' + version],
    },
  ] + [
    {
      name: 'navidrome test ' + distro,
      image: platform_image(distro, arch),
      commands: ['./navidrome/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'backend',
      image: 'golang:' + go,
      commands: ['./backend/build.sh'],
    },
    {
      name: 'cli',
      image: 'golang:' + go,
      commands: ['./cli/build.sh'],
    },
  ] + [
    {
      name: 'cli test ' + distro,
      image: platform_image(distro, arch),
      commands: ['./cli/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'package',
      image: 'debian:bookworm-slim',
      commands: ['./package.sh ' + name + ' $DRONE_BUILD_NUMBER'],
    },
  ] + [
         {
           name: 'test ' + distro,
           image: 'python:' + python,
           commands: ['./test/ci-test.sh ' + distro + ' ' + arch],
         }
         for distro in distros
       ] + (if ui then [
         {
           name: 'test-ui-' + project,
           image: 'mcr.microsoft.com/playwright:' + playwright,
           commands: ['./web/e2e/ci-ui.sh ' + project],
         }
         for project in ['desktop']
       ] else []) + [
    {
      name: 'publish',
      image: 'syncloud/store-publisher:' + store_publisher,
      environment: {
        SYNCLOUD_TOKEN: { from_secret: 'SYNCLOUD_TOKEN' },
      },
      command: ['snap', '-c', '${DRONE_BRANCH}'],
      when: {
        branch: ['master', 'stable'],
        event: ['push'],
      },
    },
    {
      name: 'artifact',
      image: 'appleboy/drone-scp:1.6.4',
      settings: {
        host: { from_secret: 'artifact_host' },
        username: 'artifact',
        key: { from_secret: 'artifact_key' },
        timeout: '2m',
        command_timeout: '2m',
        target: '/home/artifact/repo/' + name + '/${DRONE_BUILD_NUMBER}-' + arch,
        source: 'artifact/*',
        strip_components: 1,
      },
      when: {
        status: ['failure', 'success'],
        event: ['push'],
      },
    },
  ],
  trigger: {
    event: ['push'],
  },
  services: [
    {
      name: name + '.' + distro + '.com',
      image: platform_image(distro, arch),
      privileged: true,
      entrypoint: ['/bin/sh', '-c', "mkdir -p /etc/systemd/system/snapd.service.d && printf '[Service]\\nExecStartPost=/bin/sh -c \"/usr/bin/snap set system refresh.hold=2099-01-01T00:00:00Z\"\\n' > /etc/systemd/system/snapd.service.d/disable-refresh.conf && exec /sbin/init"],
      volumes: [
        { name: 'dbus', path: '/var/run/dbus' },
        { name: 'dev', path: '/dev' },
      ],
    }
    for distro in distros
  ],
  volumes: [
    { name: 'dbus', host: { path: '/var/run/dbus' } },
    { name: 'dev', host: { path: '/dev' } },
  ],
}];

build('amd64', true) +
build('arm64', false)
