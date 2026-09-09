FROM node:20-alpine
WORKDIR /app

# Install deps first (better caching). Use npm install since we don't ship a lockfile
# in this teaching scaffold; add `npm ci` + package-lock.json for reproducible builds.
COPY package.json ./
RUN npm install --omit=dev && npm cache clean --force

COPY . .

# node:alpine ships a non-root `node` user.
USER node

EXPOSE 3000
CMD ["node", "server.js"]
