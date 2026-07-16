const test = require('node:test');
const assert = require('node:assert/strict');

const { buildLoginURL, escapeAttribute, highlightParts, initialMode, isActiveRequest, isSafeLinkHref, safeReturnPath, shouldAutoExpand, shouldLoadTimeline } = require('./app-core.js');

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
        buildLoginURL('/', '?month=2026-07'),
        '/login.html?return=%2F%3Fmonth%3D2026-07',
    );
});

test('accepts only same-origin login return paths without control characters', () => {
    const origin = 'https://daily.example';
    assert.equal(safeReturnPath('/?month=2026-07', origin), '/?month=2026-07');
    assert.equal(safeReturnPath('/archive#july', origin), '/archive#july');
    for (const value of ['//evil.example', '/\\evil.example', '/\n/evil.example', '/\t/evil.example', 'https://evil.example/', 'relative']) {
        assert.equal(safeReturnPath(value, origin), '/');
    }
});
