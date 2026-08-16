import { useMemo, useState } from 'react'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '../components/ui/pagination'

export function usePagination<T>(data: T[], perPage = 10) {
  const [page, setPage] = useState(1)

  const totalPages = Math.max(1, Math.ceil(data.length / perPage))

  const safePage = Math.min(page, totalPages)

  const pageItems = useMemo(() => {
    const start = (safePage - 1) * perPage
    return data.slice(start, start + perPage)
  }, [data, safePage, perPage])

  const goToPage = (p: number) => {
    setPage(Math.min(Math.max(1, p), totalPages))
  }

  const resetPage = () => setPage(1)

  return {
    page: safePage,
    totalPages,
    pageItems,
    goToPage,
    resetPage,
    hasNext: safePage < totalPages,
    hasPrev: safePage > 1,
  }
}

type PaginationControlProps = {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
  siblingCount?: number
}

function getPageRange(currentPage: number, totalPages: number, siblingCount: number): (number | 'ellipsis')[] {
  const totalNumbers = siblingCount * 2 + 5 
  if (totalPages <= totalNumbers) {
    return Array.from({ length: totalPages }, (_, i) => i + 1)
  }

  const leftSibling = Math.max(currentPage - siblingCount, 1)
  const rightSibling = Math.min(currentPage + siblingCount, totalPages)

  const showLeftEllipsis = leftSibling > 2
  const showRightEllipsis = rightSibling < totalPages - 1

  const pages: (number | 'ellipsis')[] = [1]

  if (showLeftEllipsis) pages.push('ellipsis')
  for (let i = leftSibling; i <= rightSibling; i++) {
    if (i !== 1 && i !== totalPages) pages.push(i)
  }
  if (showRightEllipsis) pages.push('ellipsis')

  pages.push(totalPages)

  return pages
}

export function PaginationControl({ currentPage, totalPages, onPageChange, siblingCount = 1 }: PaginationControlProps) {
  if (totalPages <= 1) return null

  const range = getPageRange(currentPage, totalPages, siblingCount)

  return (
    <Pagination>
      <PaginationContent>
        <PaginationItem>
          <PaginationPrevious
            href="#"
            onClick={(e) => {
              e.preventDefault()
              if (currentPage > 1) onPageChange(currentPage - 1)
            }}
            className={currentPage <= 1 ? 'pointer-events-none opacity-50' : undefined}
          />
        </PaginationItem>

        {range.map((p, idx) =>
          p === 'ellipsis' ? (
            <PaginationItem key={`ellipsis-${idx}`}>
              <PaginationEllipsis />
            </PaginationItem>
          ) : (
            <PaginationItem key={p}>
              <PaginationLink
                href="#"
                isActive={p === currentPage}
                onClick={(e) => {
                  e.preventDefault()
                  onPageChange(p)
                }}
              >
                {p}
              </PaginationLink>
            </PaginationItem>
          )
        )}

        <PaginationItem>
          <PaginationNext
            href="#"
            onClick={(e) => {
              e.preventDefault()
              if (currentPage < totalPages) onPageChange(currentPage + 1)
            }}
            className={currentPage >= totalPages ? 'pointer-events-none opacity-50' : undefined}
          />
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  )
}