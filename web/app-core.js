(function (root, factory) {
    const api = factory();
    if (typeof module === 'object' && module.exports) module.exports = api;
    if (root) root.DailyFlowCore = api;
}(typeof globalThis !== 'undefined' ? globalThis : this, function () {
    const INITIAL_EXPANDED_COUNT = 7;

    function shouldAutoExpand(mode, renderedCount) {
        return mode === 'timeline' && renderedCount < INITIAL_EXPANDED_COUNT;
    }

    function escapeAttribute(value) {
        return String(value)
            .replaceAll('&', '&amp;')
            .replaceAll('"', '&quot;')
            .replaceAll("'", '&#39;')
            .replaceAll('<', '&lt;')
            .replaceAll('>', '&gt;');
    }

    function isSafeLinkHref(value) {
        if (typeof value !== 'string') return false;
        const href = value.trim();
        if (!href || href.includes('\\')) return false;
        if (/^https?:\/\//i.test(href) || /^mailto:/i.test(href)) return true;
        if (href.startsWith('//')) return false;
        if (href.startsWith('/') || href.startsWith('./') || href.startsWith('../') || href.startsWith('#') || href.startsWith('?')) return true;
        const boundary = href.search(/[/?#]/);
        const prefix = boundary === -1 ? href : href.slice(0, boundary);
        return !/[:&\u0000-\u0020\u007f-\u009f]/.test(prefix);
    }

    function highlightParts(text, query) {
        if (!query) return [{ text, match: false }];
        const parts = [];
        const lowerText = text.toLocaleLowerCase();
        const lowerQuery = query.toLocaleLowerCase();
        let offset = 0;
        let matchIndex;
        while ((matchIndex = lowerText.indexOf(lowerQuery, offset)) !== -1) {
            if (matchIndex > offset) parts.push({ text: text.slice(offset, matchIndex), match: false });
            const end = matchIndex + query.length;
            parts.push({ text: text.slice(matchIndex, end), match: true });
            offset = end;
        }
        if (offset < text.length) parts.push({ text: text.slice(offset), match: false });
        return parts;
    }

    function initialMode() {
        return 'routing';
    }

    function isActiveRequest(currentMode, expectedMode, requestGeneration, currentGeneration) {
        return currentMode === expectedMode && requestGeneration === currentGeneration;
    }

    function shouldLoadTimeline(mode) {
        return mode === 'timeline';
    }

    function isValidEntryPath(path) {
        if (typeof path !== 'string' || !path.startsWith('/') || !path.toLowerCase().endsWith('.md') || path.includes('\\')) {
            return false;
        }
        const segments = path.slice(1).split('/');
        return segments.every(segment => segment && segment !== '.' && segment !== '..');
    }

    function safeReturnPath(value, origin) {
        if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//') || value.includes('\\') || /[\u0000-\u001f\u007f]/.test(value)) {
            return '/';
        }
        try {
            const base = new URL(origin);
            const target = new URL(value, base);
            if (target.origin !== base.origin) return '/';
            return `${target.pathname}${target.search}${target.hash}`;
        } catch {
            return '/';
        }
    }

    function buildLoginURL(pathname, search = '') {
        return `/login.html?return=${encodeURIComponent(pathname + search)}`;
    }

    function buildEntryURL(path, baseHref) {
        if (!isValidEntryPath(path)) throw new Error('Invalid entry path');
        const url = new URL(baseHref);
        url.search = '';
        url.hash = '';
        url.searchParams.set('entry', path);
        return url.toString();
    }

    return { buildEntryURL, buildLoginURL, escapeAttribute, highlightParts, initialMode, isActiveRequest, isSafeLinkHref, isValidEntryPath, safeReturnPath, shouldAutoExpand, shouldLoadTimeline };
}));
