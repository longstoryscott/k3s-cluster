export const formatDate = (date: Date, format: string): string => {
  const options: Intl.DateTimeFormatOptions = {};

  if (format.includes('YYYY')) {
    options.year = 'numeric';
  }
  if (format.includes('MM')) {
    options.month = '2-digit';
  }
  if (format.includes('DD')) {
    options.day = '2-digit';
  }
  if (format.includes('HH')) {
    options.hour = '2-digit';
    options.hour12 = false;
  }
  if (format.includes('mm')) {
    options.minute = '2-digit';
  }
  if (format.includes('ss')) {
    options.second = '2-digit';
  }

  return new Intl.DateTimeFormat('en-US', options).format(date);
};

export const formatRelativeDate = (date: Date): string => {
  const now = new Date();
  const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (diffInSeconds < 60) {
    return `${diffInSeconds} seconds ago`;
  } else if (diffInSeconds < 3600) {
    const minutes = Math.floor(diffInSeconds / 60);
    return `${minutes} minutes ago`;
  } else if (diffInSeconds < 86400) {
    const hours = Math.floor(diffInSeconds / 3600);
    return `${hours} hours ago`;
  } else {
    return formatDate(date, 'YYYY-MM-DD');
  }
};