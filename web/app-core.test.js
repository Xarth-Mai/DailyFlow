const test = require('node:test');
const assert = require('node:assert/strict');

const { buildEntryURL, buildLoginURL, escapeAttribute, highlightParts, initialMode, isActiveRequest, isSafeLinkHref, isValidEntryPath, safeReturnPath, shouldAutoExpand, shouldLoadTimeline } = require('./app-core.js');

test('starts in routing mode so observers cannot race the initial route', () => {
    assert.equal(initialMode(), 'routing');
    assert.equal(shouldLoadTimeline(initialMode()), false);
    assert.equal(shouldLoadTimeline('timeline'), true);
});

test('accepts responses only for the current mode and generation', () => {
    assert.equal(isActiveRequest('timeline', 'timeline', 3, 3), true);
    assert.equal(isActiveRequest('search', 'timeline', 3, 3), false);
    assert.equal(isActiveRequest('timeline', 'timeline', 2, 3), false);
});

test('auto-expands only the first seven timeline entries', () => {
    for (let index = 0; index < 7; index += 1) {
        assert.equal(shouldAutoExpand('timeline', index), true);
    }
    assert.equal(shouldAutoExpand('timeline', 7), false);
});

test('does not auto-expand search results', () => {
    assert.equal(shouldAutoExpand('search', 0), false);
});

test('splits case-insensitive search matches without producing HTML', () => {
    assert.deepEqual(highlightParts('Sunny day, sunny mood', 'SUNNY'), [
        { text: 'Sunny', match: true },
        { text: ' day, ', match: false },
        { text: 'sunny', match: true },
        { text: ' mood', match: false },
    ]);
});

test('escapes values before inserting them into HTML attributes', () => {
    assert.equal(
        escapeAttribute('" onerror="alert(1)&<>\''),
        '&quot; onerror=&quot;alert(1)&amp;&lt;&gt;&#39;',
    );
});

test('allows only safe Markdown link destinations', () => {
    for (const href of ['entry.md', '../entry.md', '/archive', '#section', '?month=2026-07', 'https://example.com', 'http://example.com', 'mailto:test@example.com']) {
        assert.equal(isSafeLinkHref(href), true, href);
    }
    for (const href of ['javascript:alert(1)', 'jav&#x61;script:alert(1)', 'javascript&#x3a;alert(1)', 'data:text/html,x', 'vbscript:msgbox(1)', '//evil.example', 'java\nscript:alert(1)']) {
        assert.equal(isSafeLinkHref(href), false, href);
    }
});

test('preserves the current page when redirecting an expired session to login', () => {
    assert.equal(
        buildLoginURL('/', '?entry=%2F2026%2F07%2Fentry.md'),
        '/login.html?return=%2F%3Fentry%3D%252F2026%252F07%252Fentry.md',
    );
});

test('accepts only same-origin login return paths without control characters', () => {
    const origin = 'https://daily.example';
    assert.equal(safeReturnPath('/?entry=%2F2026%2F07%2Fentry.md', origin), '/?entry=%2F2026%2F07%2Fentry.md');
    assert.equal(safeReturnPath('/archive#july', origin), '/archive#july');
    for (const value of ['//evil.example', '/\\evil.example', '/\n/evil.example', '/\t/evil.example', 'https://evil.example/', 'relative']) {
        assert.equal(safeReturnPath(value, origin), '/');
    }
});

test('builds stable links only for safe Markdown entry paths', () => {
    assert.equal(isValidEntryPath('/2026/07/2026-07-17.md'), true);
    assert.equal(isValidEntryPath('/2026/07/Entry.MD'), true);
    assert.equal(isValidEntryPath('/../../etc/passwd'), false);
    assert.equal(isValidEntryPath('https://example.com/entry.md'), false);
    assert.equal(isValidEntryPath('/notes.txt'), false);
    assert.equal(
        buildEntryURL('/2026/07/2026-07-17.md', 'https://daily.example/?month=2026-07#old'),
        'https://daily.example/?entry=%2F2026%2F07%2F2026-07-17.md',
    );
});
