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

    const autoFocusQuery = window.matchMedia('(hover: none), (pointer: coarse)');
    let focusFrame = null;

    function updateTimelineFocus() {
        focusFrame = null;
        const items = Array.from(timelineEl.querySelectorAll('.timeline-item'));
        if (!autoFocusQuery.matches || items.length === 0) {
            items.forEach(item => item.classList.remove('is-focused'));
            return;
        }
        const viewportCenter = window.innerHeight / 2;
        let focusedItem = null;
        let closestDistance = Infinity;
        items.forEach(item => {
            const rect = item.getBoundingClientRect();
            const distance = viewportCenter < rect.top
                ? rect.top - viewportCenter
                : viewportCenter > rect.bottom ? viewportCenter - rect.bottom : 0;
            if (distance < closestDistance) {
                closestDistance = distance;
                focusedItem = item;
            }
        });
        items.forEach(item => item.classList.toggle('is-focused', item === focusedItem));
    }

    function scheduleTimelineFocus() {
        if (focusFrame === null) focusFrame = window.requestAnimationFrame(updateTimelineFocus);
    }

    window.addEventListener('scroll', scheduleTimelineFocus, { passive: true });
    window.addEventListener('resize', scheduleTimelineFocus);
    autoFocusQuery.addEventListener('change', scheduleTimelineFocus);

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
            const responseHasMore = response.headers.get('X-Has-More') === 'true';
            const entries = await response.json();
            if (!DailyFlowCore.isActiveRequest(currentMode, 'timeline', generation, requestGeneration)) return;
            if (!entries || entries.length === 0) {
                hasMore = false;
                sentinelEl.style.display = 'none';
                if (page === 1) showMessage('No entries found.');
                return;
            }
            renderEntries(entries);
            hasMore = responseHasMore;
            sentinelEl.style.display = hasMore ? 'flex' : 'none';
            currentPage++;
        } catch (error) {
            if (error.name !== 'AbortError' && DailyFlowCore.isActiveRequest(currentMode, 'timeline', generation, requestGeneration)) {
                console.error(error);
                hasMore = false;
                sentinelEl.style.display = 'none';
                showMessage('Could not load entries.');
            }
        } finally {
            if (activeRequestController === controller) {
                activeRequestController = null;
                isLoading = false;
            }
        }
    }

    function renderEntries(entries) {
        entries.forEach(entry => {
            const autoExpand = DailyFlowCore.shouldAutoExpand(currentMode, renderedTimelineCount);
            renderTimelineItem(entry.path, entry.content, autoExpand);
            renderedTimelineCount++;
        });
    }

    function renderTimelineItem(path, markdown, autoExpand = false) {
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
                scheduleTimelineFocus();
            });
            cardEl.append(contentWrapper, button);
        } else {
            contentWrapper.innerHTML = marked.parse(markdown);
            cardEl.appendChild(contentWrapper);
        }

        itemEl.appendChild(cardEl);
        timelineEl.appendChild(itemEl);
        scheduleTimelineFocus();
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

        const expandButton = document.createElement('button');
        expandButton.type = 'button';
        expandButton.className = 'read-more-btn';
        expandButton.textContent = 'Read More';
        expandButton.addEventListener('click', () => expandSearchResult(result.path, cardEl, expandButton));
        cardEl.appendChild(expandButton);
        itemEl.appendChild(cardEl);
        timelineEl.appendChild(itemEl);
        scheduleTimelineFocus();
    }

    async function expandSearchResult(path, cardEl, button) {
        const generation = requestGeneration;
        button.disabled = true;
        try {
            const response = await fetch(`/api/entry?path=${encodeURIComponent(path)}`);
            if (redirectIfUnauthorized(response)) return;
            if (!response.ok) throw new Error(`Entry request failed: ${response.status}`);
            const markdown = await response.text();
            if (!DailyFlowCore.isActiveRequest(currentMode, 'search', generation, requestGeneration)) return;
            currentMarkdownPath = path;
            const content = document.createElement('div');
            content.className = 'markdown-content';
            content.innerHTML = marked.parse(markdown);
            cardEl.replaceChildren(content);
            scheduleTimelineFocus();
        } catch (error) {
            console.error(error);
            button.disabled = false;
            button.textContent = 'Try Again';
        }
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
