# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package configs
COPY package*.json ./

# Install packages
RUN npm install

# Copy source code
COPY . .

# Build assets
RUN npm run build

# Run stage
FROM nginx:alpine

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
