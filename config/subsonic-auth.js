// Turn Subsonic query credentials (u + p, plaintext or enc:hex) into an
// HTTP Basic Authorization header so the platform Authelia (LDAP) can validate
// them. A real Authorization header (client already using HTTP Basic) is passed
// through unchanged. Token auth (t/s) yields an empty value -> Authelia denies,
// which is correct: token auth cannot be validated against an external store.
function subsonicBasic(r) {
    var existing = r.headersIn['Authorization'];
    if (existing) {
        return existing;
    }
    var u = r.args.u;
    var p = r.args.p;
    if (!u || !p) {
        return '';
    }
    if (p.substr(0, 4) === 'enc:') {
        p = Buffer.from(p.substr(4), 'hex').toString('utf8');
    }
    return 'Basic ' + Buffer.from(u + ':' + p).toString('base64');
}

export default { subsonicBasic };
