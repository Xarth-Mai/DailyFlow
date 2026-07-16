document.addEventListener('DOMContentLoaded', () => {
    let currentPage = 1;
    let isLoading = false;
    let hasMore = true;
    let currentMode = DailyFlowCore.initialMode();
    let currentMonth = '';
    let renderedTimelineCount = 0;
    let requestGeneration = 0;
    let activeRequestController = null;
    let searchTimeout;

    function beginMode(mode) {
        clearTimeout(searchTimeout);
        searchTimeout = null;
        requestGeneration++;
        currentMode = mode;
        if (activeRequestController) activeRequestController.abort();
        activeRequestController = null;
        isLoading = false;
        return requestGeneration;
    }

    const timelineEl = document.getElementById('timeline');
    const sentinelEl = document.getElementById('loading-sentinel');
    const searchInput = document.getElementById('search-input');
    const monthSelect = document.getElementById('month-select');
    const logoutBtn = document.getElementById('logout-btn');

    // Theme logic
    const themeBtn = document.getElementById('theme-btn');
    const themeIcon = document.getElementById('theme-icon');
    const THEMES = ['system', 'light', 'dark'];
    const systemDarkQuery = window.matchMedia('(prefers-color-scheme: dark)');

    function updateThemeIcon(theme) {
        if (theme === 'system') {
            themeIcon.innerHTML = `<path d="M21 9C19.3529 9 17.8917 9.79647 16.9808 11.0253M21 6C18.5797 6 16.4104 7.07479 14.9434 8.77313M21 3C17.1326 3 13.7313 4.99586 11.77 8.01376M11.77 8.01376C9.72698 8.16181 8.00348 9.48869 7.25 11.25C4.7 11.6562 3 13.7572 3 16.0315C3 18.7755 5.28335 21 8.1 21L15.75 21C18.0972 21 20 19.1279 20 16.8185C20 15.1039 18.951 13.5202 17.45 12.875C17.4116 12.2181 17.2475 11.5941 16.9808 11.0253M11.77 8.01376C11.8958 8.00465 12.0229 8 12.151 8C13.1755 8 14.1323 8.28298 14.9434 8.77313M14.9434 8.77313C15.8305 9.30914 16.5435 10.0929 16.9808 11.0253" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>`;
        } else if (theme === 'light') {
            themeIcon.innerHTML = '<circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>';
        } else {
            themeIcon.innerHTML = '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>';
        }
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute(
            'data-theme',
            theme === 'system' ? (systemDarkQuery.matches ? 'dark' : 'light') : theme,
        );
        localStorage.setItem('dailyflow-theme', theme);
        updateThemeIcon(theme);
    }

    systemDarkQuery.addEventListener('change', () => {
        if (localStorage.getItem('dailyflow-theme') === 'system') applyTheme('system');
    });
    applyTheme(localStorage.getItem('dailyflow-theme') || 'system');
    themeBtn.addEventListener('click', () => {
        const currentTheme = localStorage.getItem('dailyflow-theme') || 'system';
        applyTheme(THEMES[(THEMES.indexOf(currentTheme) + 1) % THEMES.length]);
    });

    // Marked renderer
    const renderer = new marked.Renderer();
    let currentMarkdownPath = '';
    renderer.image = function (token) {
        let href = token.href || '';
        const title = token.title || '';
        const text = token.text || '';
        if (href && !href.startsWith('http') && !href.startsWith('/')) {
            const dir = currentMarkdownPath.substring(0, currentMarkdownPath.lastIndexOf('/'));
            if (href.startsWith('./')) href = href.substring(2);
            href = `/raw${dir}/${href}`;
        }
        return `<img src="${DailyFlowCore.escapeAttribute(href)}" alt="${DailyFlowCore.escapeAttribute(text)}" title="${DailyFlowCore.escapeAttribute(title)}">`;
    };
    renderer.html = token => DailyFlowCore.escapeAttribute(token.text || '');
    renderer.link = function (token) {
        const label = this.parser.parseInline(token.tokens || []);
        if (!DailyFlowCore.isSafeLinkHref(token.href)) return label;
        const title = token.title ? ` title="${DailyFlowCore.escapeAttribute(token.title)}"` : '';
        return `<a href="${DailyFlowCore.escapeAttribute(token.href)}"${title}>${label}</a>`;
    };
    marked.setOptions({ renderer });

    function redirectIfUnauthorized(response) {
        if (response.status !== 401) return false;
        window.location.href = DailyFlowCore.buildLoginURL(window.location.pathname, window.location.search);
        return true;
    }

    function resetTimeline() {
        timelineEl.innerHTML = '';
        currentPage = 1;
        hasMore = true;
        renderedTimelineCount = 0;
    }

    function showMessage(message) {
        timelineEl.innerHTML = '';
        const element = document.createElement('p');
        element.className = 'empty-msg';
        element.textContent = message;
        timelineEl.appendChild(element);
    }

    async function fetchPage(page) {
        if (isLoading || !hasMore || !DailyFlowCore.shouldLoadTimeline(currentMode)) return;
        const generation = requestGeneration;
        const controller = new AbortController();
        activeRequestController = controller;
        isLoading = true;
        try {
            const params = new URLSearchParams({ page: String(page) });
            if (currentMonth) params.set('month', currentMonth);
            const response = await fetch(`/api/list?${params}`, { signal: controller.signal });
            if (redirectIfUnauthorized(response)) return;
            if (!response.ok) throw new Error(`List request failed: ${response.status}`);
            const entries = await response.json();
            if (!DailyFlowCore.isActiveRequest(currentMode, 'timeline', generation, requestGeneration)) return;
            if (!entries || entries.length === 0) {
                hasMore = false;
                sentinelEl.style.display = 'none';
                if (page === 1) showMessage('No entries found.');
                return;
            }
            renderEntries(entries);
            currentPage++;
        } catch (error) {
            if (error.name !== 'AbortError' && DailyFlowCore.isActiveRequest(currentMode, 'timeline', generation, requestGeneration)) {
                console.error(error);
                showMessage('Could not load entries.');
            }
        } finally {
            if (activeRequestController === controller) {
                activeRequestController = null;
                isLoading = false;
            }
        }
    }

    function createActionButton(label, handler) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'card-action-btn';
        button.textContent = label;
        button.addEventListener('click', handler);
        return button;
    }

    async function copyEntryLink(path, button) {
        try {
            await navigator.clipboard.writeText(DailyFlowCore.buildEntryURL(path, window.location.href));
            button.textContent = 'Copied';
        } catch (error) {
            console.error(error);
            button.textContent = 'Copy failed';
        }
        window.setTimeout(() => { button.textContent = 'Copy Link'; }, 1500);
    }

    function appendEntryActions(cardEl, path, includeBack = false) {
        const actions = document.createElement('div');
        actions.className = 'card-actions';
        const copyButton = createActionButton('Copy Link', () => copyEntryLink(path, copyButton));
        actions.appendChild(copyButton);
        if (includeBack) {
            actions.appendChild(createActionButton('Back to Timeline', () => {
                history.pushState({}, '', window.location.pathname);
                showTimeline('');
            }));
        }
        cardEl.appendChild(actions);
    }

    function renderEntries(entries) {
        entries.forEach(entry => {
            const autoExpand = DailyFlowCore.shouldAutoExpand(currentMode, renderedTimelineCount);
            renderTimelineItem(entry.path, entry.content, autoExpand);
            renderedTimelineCount++;
        });
    }

    function renderTimelineItem(path, markdown, autoExpand = false, includeBack = false) {
        const itemEl = document.createElement('div');
        itemEl.className = 'timeline-item';

        const dateEl = document.createElement('div');
        dateEl.className = 'timeline-date';
        dateEl.textContent = path.split('/').pop().replace(/\.md$/i, '');
        itemEl.appendChild(dateEl);

        const cardEl = document.createElement('div');
        cardEl.className = 'timeline-card';
        const contentWrapper = document.createElement('div');
        contentWrapper.className = 'markdown-content';

        currentMarkdownPath = path;
        const lines = markdown.split('\n');
        const needsExpansion = lines.filter(line => line.trim() !== '').length > 4;
        if (needsExpansion && !autoExpand) {
            contentWrapper.innerHTML = marked.parse(lines.slice(0, 5).join('\n'));
            const button = document.createElement('button');
            button.type = 'button';
            button.className = 'read-more-btn';
            button.textContent = 'Read More';
            button.addEventListener('click', () => {
                currentMarkdownPath = path;
                contentWrapper.innerHTML = marked.parse(markdown);
                button.remove();
            });
            cardEl.append(contentWrapper, button);
        } else {
            contentWrapper.innerHTML = marked.parse(markdown);
            cardEl.appendChild(contentWrapper);
        }

        appendEntryActions(cardEl, path, includeBack);
        itemEl.appendChild(cardEl);
        timelineEl.appendChild(itemEl);
    }

    function appendHighlightedText(element, text, query) {
        DailyFlowCore.highlightParts(text, query).forEach(part => {
            const node = part.match ? document.createElement('mark') : document.createTextNode(part.text);
            if (part.match) node.textContent = part.text;
            element.appendChild(node);
        });
    }

    function renderSearchResult(result, query) {
        const itemEl = document.createElement('div');
        itemEl.className = 'timeline-item';

        const dateEl = document.createElement('div');
        dateEl.className = 'timeline-date';
        dateEl.textContent = result.path.split('/').pop().replace(/\.md$/i, '');
        itemEl.appendChild(dateEl);

        const cardEl = document.createElement('div');
        cardEl.className = 'timeline-card';
        const titleEl = document.createElement('h2');
        titleEl.className = 'search-result-title';
        titleEl.textContent = result.title;
        const snippetEl = document.createElement('p');
        snippetEl.className = 'search-snippet';
        appendHighlightedText(snippetEl, result.snippet, query);
        cardEl.append(titleEl, snippetEl);

        const actions = document.createElement('div');
        actions.className = 'card-actions';
        actions.appendChild(createActionButton('Open Entry', () => showEntry(result.path, true)));
        const copyButton = createActionButton('Copy Link', () => copyEntryLink(result.path, copyButton));
        actions.appendChild(copyButton);
        cardEl.appendChild(actions);
        itemEl.appendChild(cardEl);
        timelineEl.appendChild(itemEl);
    }

    async function performSearch(query, generation) {
        if (!DailyFlowCore.isActiveRequest(currentMode, 'search', generation, requestGeneration)) return;
        const controller = new AbortController();
        activeRequestController = controller;
        try {
            const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`, { signal: controller.signal });
            if (redirectIfUnauthorized(response)) return;
            if (!response.ok) throw new Error(`Search request failed: ${response.status}`);
            const results = await response.json();
            if (!DailyFlowCore.isActiveRequest(currentMode, 'search', generation, requestGeneration)) return;
            if (!results || results.length === 0) {
                showMessage('No results found.');
                return;
            }
            results.forEach(result => renderSearchResult(result, query));
        } catch (error) {
            if (error.name !== 'AbortError' && DailyFlowCore.isActiveRequest(currentMode, 'search', generation, requestGeneration)) {
                console.error(error);
                showMessage('Search failed.');
            }
        } finally {
            if (activeRequestController === controller) activeRequestController = null;
        }
    }

    async function showEntry(path, pushHistory = false) {
        const generation = beginMode('entry');
        if (!DailyFlowCore.isValidEntryPath(path)) {
            sentinelEl.style.display = 'none';
            showMessage('Invalid entry link.');
            return;
        }
        if (pushHistory) history.pushState({}, '', DailyFlowCore.buildEntryURL(path, window.location.href));
        timelineEl.innerHTML = '';
        sentinelEl.style.display = 'none';
        const controller = new AbortController();
        activeRequestController = controller;
        try {
            const response = await fetch(`/api/entry?path=${encodeURIComponent(path)}`, { signal: controller.signal });
            if (redirectIfUnauthorized(response)) return;
            if (!response.ok) throw new Error(`Entry request failed: ${response.status}`);
            const markdown = await response.text();
            if (!DailyFlowCore.isActiveRequest(currentMode, 'entry', generation, requestGeneration)) return;
            renderTimelineItem(path, markdown, true, true);
        } catch (error) {
            if (error.name !== 'AbortError' && DailyFlowCore.isActiveRequest(currentMode, 'entry', generation, requestGeneration)) {
                console.error(error);
                showMessage('Could not load this entry.');
            }
        } finally {
            if (activeRequestController === controller) activeRequestController = null;
        }
    }

    function updateTimelineURL(month) {
        const url = new URL(window.location.href);
        url.search = '';
        url.hash = '';
        if (month) url.searchParams.set('month', month);
        history.pushState({}, '', url);
    }

    function showTimeline(month, updateHistory = false) {
        beginMode('timeline');
        currentMonth = month;
        monthSelect.value = month;
        searchInput.value = '';
        resetTimeline();
        sentinelEl.style.display = 'flex';
        if (updateHistory) updateTimelineURL(month);
        fetchPage(1);
    }

    async function loadMonths() {
        const response = await fetch('/api/months');
        if (redirectIfUnauthorized(response)) return;
        if (!response.ok) throw new Error(`Months request failed: ${response.status}`);
        const months = await response.json();
        months.forEach(month => {
            const option = document.createElement('option');
            option.value = month;
            option.textContent = month;
            monthSelect.appendChild(option);
        });
    }

    searchInput.addEventListener('input', event => {
        clearTimeout(searchTimeout);
        const query = event.target.value.trim();
        if (!query) {
            showTimeline(currentMonth);
            return;
        }
        const generation = beginMode('search');
        timelineEl.innerHTML = '';
        sentinelEl.style.display = 'none';
        searchTimeout = window.setTimeout(() => performSearch(query, generation), 350);
    });

    monthSelect.addEventListener('change', event => showTimeline(event.target.value, true));
    logoutBtn.addEventListener('click', async () => {
        const response = await fetch('/api/logout', { method: 'POST' });
        if (response.ok) window.location.href = '/login.html';
    });

    const observer = new IntersectionObserver(entries => {
        if (entries[0].isIntersecting && !isLoading && hasMore && DailyFlowCore.shouldLoadTimeline(currentMode)) {
            fetchPage(currentPage);
        }
    }, { rootMargin: '200px' });
    observer.observe(sentinelEl);

    async function routeFromLocation() {
        const params = new URLSearchParams(window.location.search);
        const entry = params.get('entry');
        if (entry) {
            await showEntry(entry);
            return;
        }
        const requestedMonth = params.get('month') || '';
        const monthExists = requestedMonth === '' || Array.from(monthSelect.options).some(option => option.value === requestedMonth);
        showTimeline(monthExists ? requestedMonth : '');
    }

    window.addEventListener('popstate', routeFromLocation);
    loadMonths().then(routeFromLocation).catch(error => {
        console.error(error);
        showMessage('Could not load journal navigation.');
    });
});
