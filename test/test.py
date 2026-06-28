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


def _diag(device):
    print("\n========== BACKEND JOURNAL ==========")
    print(device.run_ssh('journalctl -u snap.navidrome.backend --no-pager | tail -120', throw=False))
    print("\n========== NAVIDROME JOURNAL ==========")
    print(device.run_ssh('journalctl -u snap.navidrome.navidrome --no-pager | tail -120', throw=False))
    print("\n========== DIRECT navidrome.sock + Remote-User ==========")
    print(device.run_ssh(
        'curl -s -m 10 -H "Remote-User: user" --unix-socket /var/snap/navidrome/current/navidrome.sock '
        '"http://localhost/rest/ping?v=1.16.1&c=diag&f=json"', throw=False))
    print("\n========== DIRECT navidrome.sock native u/p ==========")
    print(device.run_ssh(
        'curl -s -m 10 --unix-socket /var/snap/navidrome/current/navidrome.sock '
        '"http://localhost/rest/ping?u=user&p=Password1&v=1.16.1&c=diag&f=json"', throw=False))
    print("\n========== LDAP bind probe (ldapwhoami) ==========")
    print(device.run_ssh(
        'ldapwhoami -x -H ldap://localhost:389 -D "cn=user,ou=users,dc=syncloud,dc=org" -w Password1 2>&1 || echo "ldapwhoami failed/absent"', throw=False))


@pytest.fixture(scope="session")
def module_setup(request, device, artifact_dir):
    def module_teardown():
        _diag(device)
        device.run_ssh('ls -la /var/snap/{0}/current > {1}/var.current.ls.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('cat /var/snap/{0}/current/config/nginx.conf > {1}/nginx.conf.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('cat /var/snap/{0}/current/config/oidc.env > {1}/oidc.env.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('journalctl -u snap.{0}.navidrome --no-pager | tail -2000 > {1}/navidrome.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('journalctl -u snap.{0}.backend --no-pager | tail -2000 > {1}/backend.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('journalctl -u snap.{0}.nginx --no-pager | tail -2000 > {1}/nginx.log'.format(APP, TMP_DIR), throw=False)
        device.run_ssh('journalctl --no-pager | tail -3000 > {0}/journalctl.log'.format(TMP_DIR), throw=False)
        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


def subsonic_ping_ok(app_domain, user, password):
    session = requests.session()
    url = "https://{0}/rest/ping".format(app_domain)
    params = {'u': user, 'p': password, 'v': '1.16.1', 'c': 'syncloud-test', 'f': 'json'}
    last = None
    for _ in range(60):
        try:
            r = session.get(url, params=params, verify=False, timeout=10)
            last = r.text
            if r.status_code == 200:
                resp = r.json().get('subsonic-response', {})
                if resp.get('status') == 'ok':
                    return True
        except Exception as e:
            last = str(e)
        time.sleep(2)
    return False, last


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
    device.run_ssh('test -S /var/snap/navidrome/current/backend.sock', retries=30)


def test_web_redirects_to_sso(app_domain):
    session = requests.session()
    last = None
    for _ in range(60):
        r = session.get("https://{0}/".format(app_domain), verify=False,
                        allow_redirects=False, headers={'Accept': 'text/html'}, timeout=10)
        last = r.status_code
        if r.status_code in (302, 303):
            location = r.headers.get('Location', '')
            assert '/syncloud-oidc/login' in location or 'auth.' in location, location
            return
        time.sleep(2)
    assert False, "expected redirect to SSO, last status {0}".format(last)


def test_subsonic_login_via_ldap(app_domain, device_user, device_password):
    result = subsonic_ping_ok(app_domain, device_user, device_password)
    assert result is True, "subsonic ping with syncloud credentials failed: {0}".format(result)


def test_subsonic_rejects_wrong_password(app_domain, device_user):
    session = requests.session()
    url = "https://{0}/rest/ping".format(app_domain)
    params = {'u': device_user, 'p': 'definitely-wrong', 'v': '1.16.1', 'c': 'syncloud-test', 'f': 'json'}
    r = session.get(url, params=params, verify=False, timeout=10)
    assert r.status_code == 200, r.text
    assert r.json().get('subsonic-response', {}).get('status') == 'failed', r.text


def test_remove(device, app):
    response = device.app_remove(app)
    assert response.status_code == 200, response.text


def test_reinstall(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)


def test_subsonic_after_reinstall(app_domain, device_user, device_password):
    result = subsonic_ping_ok(app_domain, device_user, device_password)
    assert result is True, "subsonic ping after reinstall failed: {0}".format(result)
