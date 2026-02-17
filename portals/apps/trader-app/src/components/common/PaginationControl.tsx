import { Button, Flex, Text } from '@radix-ui/themes'
import { ChevronLeftIcon, ChevronRightIcon } from '@radix-ui/react-icons'

interface PaginationControlProps {
    currentPage: number
    totalPages: number
    onPageChange: (page: number) => void
    hasNext: boolean
    hasPrev: boolean
    totalCount?: number
}

export function PaginationControl({
    currentPage,
    totalPages,
    onPageChange,
    hasNext,
    hasPrev,
    totalCount
}: PaginationControlProps) {
    return (
        <Flex justify="between" align="center" mt="4" pt="4" pb="4" pl="4" pr="4" style={{ borderTop: '1px solid var(--gray-5)' }}>
            <Flex align="center" gap="4">
                {totalCount !== undefined && (
                    <Text size="2" color="gray">
                        Total: {totalCount}
                    </Text>
                )}


            </Flex>

            <Flex gap="2" align="center">
                <Text size="2" color="gray" mr="2">
                    Page {currentPage} of {totalPages || 1}
                </Text>
                <Button
                    variant="soft"
                    disabled={!hasPrev}
                    onClick={() => onPageChange(currentPage - 1)}
                >
                    <ChevronLeftIcon />
                    Previous
                </Button>
                <Button
                    variant="soft"
                    disabled={!hasNext}
                    onClick={() => onPageChange(currentPage + 1)}
                >
                    Next
                    <ChevronRightIcon />
                </Button>
            </Flex>
        </Flex>
    )
}
