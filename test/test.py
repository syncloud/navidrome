import time
from subprocess import check_output

import pytest
import requests
from requests.packages.urllib3.exceptions import InsecureRequestWarning
from syncloudlib.integration.hosts import add_host_alias
from syncloudlib.integration.installer import local_install

TMP_DIR = '/tmp/syncloud'
APP = 'navidrome'

requests.packages.urllib3.disable_warnings(InsecureRequestWarning)


@pytest.fixture(scope="session")
def module_setup(request, device, artifact_dir):
    def module_teardown():
        device.run_ssh('ls -la /var/snap/{0}/current > {1}/var.current.ls.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('cat /var/snap/{0}/current/config/nginx.conf > {1}/nginx.conf.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('journalctl -u snap.{0}.navidrome --no-pager | tail -2000 > {1}/navidrome.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('journalctl -u snap.{0}.nginx --no-pager | tail -2000 > {1}/nginx.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('journalctl --no-pager | tail -3000 > {0}/journalctl.log'.format(TMP_DIR), throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


def provision_user(app_domain, user, password):
    session = requests.session()
    for _ in range(30):
        r = session.get("https://{0}/app/".format(app_domain), auth=(user, password), verify=False, timeout=10)
        if r.status_code == 200:
            return True
        time.sleep(2)
    return False


def subsonic_ping(app_domain, user, password):
    session = requests.session()
    return session.get(
        "https://{0}/rest/ping".format(app_domain),
        params={'v': '1.16.1', 'c': 'syncloud-test', 'f': 'json'},
        auth=(user, password), verify=False, timeout=10)


def test_start(module_setup, device, device_host, app, domain):
    add_host_alias(app, device_host, domain)
    device.run_ssh('date', retries=100)
    device.run_ssh('mkdir {0}'.format(TMP_DIR))


@pytest.mark.flaky(retries=50, delay=10)
def test_activate_device(device):
    device.run_ssh('rm -f /var/snap/platform/current/syncloud.crt', throw=False)
    response = device.activate_custom()
    assert response.status_code == 200, response.text


def test_install(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_sockets(device):
    device.run_ssh('test -S /var/snap/navidrome/current/navidrome.sock', retries=30)
    device.run_ssh('test -S /var/snap/navidrome/common/web.socket', retries=30)


def test_nextcloud_link_forced(device):
    device.run_ssh('test -L /data/navidrome/nextcloud')
    target = device.run_ssh('readlink /data/navidrome/nextcloud').strip()
    assert target == '/data/nextcloud', target


def test_web_requires_auth(app_domain):
    session = requests.session()
    last = None
    for _ in range(60):
        r = session.get("https://{0}/".format(app_domain), verify=False, allow_redirects=False, timeout=10)
        last = r.status_code
        if r.status_code in (301, 302, 303):
            assert 'auth.' in r.headers.get('Location', ''), r.headers.get('Location')
            return
        time.sleep(2)
    assert False, "expected redirect to Authelia portal, last status {0}".format(last)


@pytest.mark.flaky(retries=10, delay=6)
def test_subsonic_login_via_authelia(app_domain, device_user, device_password):
    assert provision_user(app_domain, device_user, device_password), "web provisioning failed"
    r = subsonic_ping(app_domain, device_user, device_password)
    assert r.status_code == 200, r.text
    assert r.json().get('subsonic-response', {}).get('status') == 'ok', r.text


def test_subsonic_rejects_wrong_password(app_domain, device_user):
    r = subsonic_ping(app_domain, device_user, 'definitely-wrong')
    assert r.status_code == 401, "expected 401 from authelia basic, got {0}: {1}".format(r.status_code, r.text[:200])


def test_remove(device, app):
    response = device.app_remove(app)
    assert response.status_code == 200, response.text


def test_reinstall(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


@pytest.mark.flaky(retries=10, delay=6)
def test_subsonic_after_reinstall(app_domain, device_user, device_password):
    assert provision_user(app_domain, device_user, device_password), "web provisioning failed after reinstall"
    r = subsonic_ping(app_domain, device_user, device_password)
    assert r.status_code == 200, r.text
    assert r.json().get('subsonic-response', {}).get('status') == 'ok', r.text
