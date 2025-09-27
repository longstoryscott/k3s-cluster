# SearxNG API Access Documentation

## Overview
SearxNG is now properly configured for automated web search and scraping in your K3s cluster.

## Access URLs
- **Web Interface**: http://192.168.0.71:8086/
- **Internal Cluster**: http://searxng.searxng.svc.cluster.local:8080/
- **NodePort**: http://192.168.0.71:32348/ (direct access)

## API Usage for Scrapy/Beautiful Soup

### Basic JSON Search
```python
import requests

# Simple search
response = requests.get('http://192.168.0.71:8086/search', params={
    'q': 'your search query',
    'format': 'json'
})
results = response.json()

# Access search results
for result in results['results']:
    print(f"Title: {result['title']}")
    print(f"URL: {result['url']}")
    print(f"Content: {result['content']}")
    print("---")
```

### Advanced Parameters
```python
params = {
    'q': 'python programming',
    'format': 'json',
    'categories': 'general',     # general, images, videos, news, etc.
    'engines': 'duckduckgo,bing', # Specific engines
    'lang': 'en',               # Language filter
    'pageno': 1                 # Page number
}
```

### Integration with Scrapy
```python
import scrapy
import requests

class SearchSpider(scrapy.Spider):
    name = 'search_spider'
    
    def start_requests(self):
        search_query = 'your topic'
        searxng_url = 'http://192.168.0.71:8086/search'
        
        # Get search results from SearxNG
        response = requests.get(searxng_url, params={
            'q': search_query,
            'format': 'json'
        })
        
        results = response.json()
        
        # Create Scrapy requests for each result URL
        for result in results['results']:
            yield scrapy.Request(
                url=result['url'],
                callback=self.parse_page,
                meta={'search_result': result}
            )
    
    def parse_page(self, response):
        # Process the actual webpage content
        yield {
            'search_title': response.meta['search_result']['title'],
            'url': response.url,
            'page_title': response.css('title::text').get(),
            'content': response.css('body').get()
        }
```

## Available Engines
- DuckDuckGo (primary, less restrictive)
- Bing (good for general results)
- Startpage (Google proxy, more anonymous)
- Wikipedia (for reference content)

## Response Structure
```json
{
  "query": "search term",
  "number_of_results": 30,
  "results": [
    {
      "url": "https://example.com",
      "title": "Page Title",
      "content": "Page description...",
      "engine": "duckduckgo",
      "score": 16.0,
      "category": "general",
      "publishedDate": "2025-01-01T00:00:00",
      "thumbnail": "image_url"
    }
  ],
  "suggestions": ["related", "searches"],
  "infoboxes": [...],
  "answers": [...]
}
```

## Configuration Details
- **Rate Limiting**: Disabled for automated use
- **Image Proxy**: Enabled
- **Multiple Engines**: Configured for better coverage
- **JSON Format**: Always available
- **Redis Cache**: Connected to cluster Redis for performance

## Troubleshooting
- If getting rate limited, add delays between requests
- Use different engines if one is blocked: `engines=startpage` vs `engines=duckduckgo`
- Monitor logs: `kubectl logs -n searxng deployment/searxng --tail=20`
- Health check: `curl http://192.168.0.71:8086/`

## Performance Tips
1. Use specific categories to reduce result processing time
2. Implement request caching in your application
3. Use the internal cluster URL for better performance when running inside K3s
4. Consider batching searches to avoid overwhelming the service

The service is now ready for production use with your Scrapy and Beautiful Soup projects!