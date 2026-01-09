# Simple Makefile for quick development workflow

.PHONY: update update-backend update-frontend update-ai update-llm rebuild-all test-llm clear-data clear-data-file clear-data-confirm clear-cache clear-all test-clear db-status

# Quick update workflow - build and restart everything
update: build-all rebuild-all
	@echo "✓ Full update complete!"

# Build backend
build-backend:
	@echo "🔨 Building Go backend..."
	cd go-backend && go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Build frontend  
build-frontend:
	@echo "🔨 Building Vue.js frontend..."
	cd vue-frontend && npm install && npm run build

# Build AI services
build-ai:
	@echo "🔨 Building Python AI services..."
	cd python-ai && pip install -r requirements.txt

# Build everything
build-all: build-backend build-frontend build-ai
	@echo "✅ All builds complete"

# Rebuild and restart all Docker services
rebuild-all:
	@echo "🔄 Rebuilding and restarting Docker services..."
	sudo docker compose down
	sudo docker compose build
	sudo docker compose up -d
	@echo "✅ All services restarted"

# Quick backend update
update-backend: build-backend
	@echo "🔄 Updating backend service..."
	sudo docker compose build go-backend
	sudo docker compose up -d go-backend

# Quick frontend update  
update-frontend: build-frontend
	@echo "🔄 Updating frontend service..."
	sudo docker compose build vue-frontend
	sudo docker compose up -d vue-frontend

# Quick AI services update
update-ai: 
	@echo "🔄 Updating AI services..."
	sudo docker compose build ai-service
	sudo docker compose up -d ai-service

# Test AI functionality (Pure LLM Analysis)
test-ai: build-ai
	@echo "🧪 Testing Pure LLM AI analysis functionality..."
	sudo docker compose up -d ai-service
	python test_llm_update.py

test-llm:
	@echo "🧪 Testing Pure LLM Analysis functionality..."
	@echo "Checking if AI service is running..."
	@curl -s http://localhost:8080/health | jq '.ai_service' || echo "❌ AI service not responding"
	@echo ""
	@echo "Running pure LLM analysis test..."
	cd python-ai && python test_llm_analysis.py

# Test compatibility (removed - no longer needed)
# test-compatibility:
# 	@echo "🔍 Testing LLM integration compatibility..."
# 	python3 test_compatibility.py

# Demo compatibility (removed - no longer needed)  
# demo-compatibility:
# 	@echo "🎭 Running LLM compatibility demo..."
# 	python3 demo_compatibility.py

# Test all services (Pure LLM Architecture)
test-all:
	@echo "🧪 Testing all services (Pure LLM Architecture)..."
	@echo "Backend health check:"
	@curl -s http://localhost:8080/health | jq . || echo "❌ Backend not responding"
	@echo ""
	@echo "Frontend check:"
	@curl -s http://localhost:3000 > /dev/null && echo "✅ Frontend responding" || echo "❌ Frontend not responding"
	@echo ""
	@echo "AI Service (Pure LLM) check:"
	@curl -s http://localhost:8080/api/v1/ai/stats | jq . || echo "❌ AI service not responding"
	@echo ""
	@echo "Running pure LLM tests..."
	@python3 test_llm_update.py

# Show logs
logs:
	sudo docker compose logs -f

# Show logs for specific service
logs-backend:
	sudo docker compose logs -f go-backend

logs-frontend:
	sudo docker compose logs -f vue-frontend

logs-ai:
	sudo docker compose logs -f ai-service

logs-ai:
	sudo docker compose logs -f ai-service

# Stop everything
stop:
	sudo docker compose down

# Clean everything (including volumes)
clean:
	sudo docker compose down -v
	sudo docker system prune -f

# Demo Pure LLM Analysis
demo-llm:
	@echo "🎭 Running Pure LLM Analysis demo..."
	python3 test_llm_update.py

# Setup environment for Pure LLM
setup-llm:
	@echo "🔧 Setting up Pure LLM environment..."
	@if [ ! -f .env ]; then \
		echo "Creating .env file from template..."; \
		cp .env.example .env; \
		echo "Please edit .env file and set SILICONFLOW_API_TOKEN"; \
	else \
		echo ".env file already exists"; \
	fi
	@echo "✅ Pure LLM environment setup complete"

# Show service status
status:
	@echo "📊 Service Status:"
	@sudo docker compose ps

# Clear all data from database
clear-data:
	@echo "⚠️  WARNING: This will delete ALL data from the database!"
	@echo "Press Ctrl+C to cancel, or wait 5 seconds to continue..."
	@sleep 5
	@echo "🗑️  Clearing database data..."
	@sudo docker compose exec postgres psql -U postgres -d log_analysis -c "\
		TRUNCATE TABLE analysis_results CASCADE; \
		TRUNCATE TABLE logs CASCADE; \
		SELECT 'logs' as table_name, COUNT(*) as record_count FROM logs \
		UNION ALL \
		SELECT 'analysis_results' as table_name, COUNT(*) as record_count FROM analysis_results \
		UNION ALL \
		SELECT 'alert_rules' as table_name, COUNT(*) as record_count FROM alert_rules;"
	@echo "✅ Database data cleared successfully!"

# Clear data using SQL file (alternative method)
clear-data-file:
	@echo "⚠️  WARNING: This will delete ALL data from the database!"
	@echo "Press Ctrl+C to cancel, or wait 5 seconds to continue..."
	@sleep 5
	@echo "🗑️  Copying SQL file to container and executing..."
	@sudo docker cp clear_data.sql $$(sudo docker compose ps -q postgres):/tmp/clear_data.sql
	@sudo docker compose exec postgres psql -U postgres -d log_analysis -f /tmp/clear_data.sql
	@echo "✅ Database data cleared successfully!"

# Clear data with confirmation (safer version)
clear-data-confirm:
	@echo "⚠️  This will permanently delete ALL data from the database!"
	@echo "Type 'yes' to confirm:"
	@read confirmation && [ "$$confirmation" = "yes" ] || (echo "❌ Operation cancelled" && exit 1)
	@echo "🗑️  Clearing database data..."
	@sudo docker compose exec postgres psql -U postgres -d log_analysis -c "\
		TRUNCATE TABLE analysis_results CASCADE; \
		TRUNCATE TABLE logs CASCADE; \
		SELECT 'Database cleared successfully!' as status;"
	@echo "✅ Database data cleared successfully!"

# Clear Redis cache
clear-cache:
	@echo "🗑️  Clearing Redis cache..."
	@sudo docker compose exec redis redis-cli FLUSHALL
	@echo "✅ Redis cache cleared successfully!"

# Clear all data (database + cache)
clear-all: clear-cache clear-data
	@echo "✅ All data cleared successfully!"

# Test clear data functionality
test-clear:
	@echo "🧪 Testing clear data functionality..."
	python3 test_clear_data.py

# Check database status
db-status:
	@echo "📊 Database Status:"
	@sudo docker compose exec postgres psql -U postgres -d log_analysis -c "\
		SELECT 'logs' as table_name, COUNT(*) as record_count FROM logs \
		UNION ALL \
		SELECT 'analysis_results' as table_name, COUNT(*) as record_count FROM analysis_results \
		UNION ALL \
		SELECT 'alert_rules' as table_name, COUNT(*) as record_count FROM alert_rules;"